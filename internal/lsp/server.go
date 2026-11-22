// Package lsp is Lunar's language server. It analyses the workspace with
// the same project pipeline as `lunar build`, so the editor and the compiler
// never disagree, and answers questions from the checker's symbol index.
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"lunar/internal/diag"
	"lunar/internal/project"
	"lunar/internal/source"
	"lunar/internal/version"
)

// Server is one language server session.
type Server struct {
	reader  *bufio.Reader
	writer  io.Writer
	writeMu sync.Mutex
	logger  *log.Logger

	// mu guards everything below; the message loop and the debounce timer
	// both take it.
	mu          sync.Mutex
	initialized bool
	shutdown    bool // shutdown was requested; exit ends the server
	exited      bool
	// watchFiles: the client lets the server register file watching.
	watchFiles bool
	root       string   // the first workspace folder, for short file names
	roots      []string // every workspace folder
	// projectsIn caches the lunar.config.json files found under each
	// workspace folder; a watched-file event finds them again.
	projectsIn map[string][]string
	docs       map[string]string // open buffers by absolute path
	// clientURIs is the URI the editor opened each file with, which is
	// how it is named back to the editor.
	clientURIs map[string]string
	debounce   time.Duration
	timer      *time.Timer
	dirty      bool

	progs []*project.Program // from the last analysis, one per project
	// lastProgs are the same programs by project, for the next analysis to
	// start from.
	lastProgs map[string]*project.Program
	modules   map[string]*project.Module // last module without syntax errors, by path
	// configErrors is the error last shown for each config file, so it is
	// shown once rather than on every keystroke.
	configErrors map[string]string
	// broken are the files that do not parse now, so their entries in
	// modules describe an older text.
	broken    map[string]bool
	published map[string]bool         // files whose diagnostics were last non-empty
	files     map[string]*source.File // text of files positions were mapped in
}

// NewServer creates a server speaking on stdin and stdout.
func NewServer() *Server {
	return &Server{
		reader:       bufio.NewReader(os.Stdin),
		writer:       os.Stdout,
		logger:       log.New(os.Stderr, "[lunar-lsp] ", log.LstdFlags),
		docs:         map[string]string{},
		clientURIs:   map[string]string{},
		debounce:     150 * time.Millisecond,
		modules:      map[string]*project.Module{},
		configErrors: map[string]string{},
		published:    map[string]bool{},
	}
}

// Run serves requests until the client disconnects or asks to exit.
// ErrExitWithoutShutdown is what Run returns for an exit notification
// that came without a shutdown request first: the process should exit 1.
var ErrExitWithoutShutdown = errors.New("exit without shutdown")

func (s *Server) Run() error {
	for {
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			// A malformed header loses that message, not the session.
			var bad *badHeader
			if errors.As(err, &bad) {
				s.logger.Printf("skipping a message: %v", err)
				continue
			}
			return err
		}
		if err := s.handleMessage(msg); err != nil {
			s.logger.Printf("error handling message: %v", err)
		}
		s.mu.Lock()
		exited, shutdown := s.exited, s.shutdown
		s.mu.Unlock()
		if exited {
			if !shutdown {
				return ErrExitWithoutShutdown
			}
			return nil
		}
	}
}

func (s *Server) readMessage() (json.RawMessage, error) {
	contentLength := 0
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		// After a lost message its body runs into the next header line:
		// the last Content-Length on the line is the next message's.
		if at := strings.LastIndex(line, "Content-Length:"); at >= 0 {
			if contentLength, err = strconv.Atoi(strings.TrimSpace(line[at+len("Content-Length:"):])); err != nil {
				contentLength = -1
			}
		}
	}
	if contentLength <= 0 {
		return nil, &badHeader{"missing or invalid Content-Length header"}
	}
	content := make([]byte, contentLength)
	if _, err := io.ReadFull(s.reader, content); err != nil {
		return nil, err
	}
	return json.RawMessage(content), nil
}

// badHeader is a message whose header could not be read.
type badHeader struct{ msg string }

func (e *badHeader) Error() string { return e.msg }

func (s *Server) sendMessage(msg interface{}) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := fmt.Fprintf(s.writer, "Content-Length: %d\r\n\r\n", len(content)); err != nil {
		return err
	}
	_, err = s.writer.Write(content)
	return err
}

func (s *Server) sendResponse(id interface{}, result interface{}) error {
	return s.sendMessage(ResponseMessage{Message: Message{JSONRPC: "2.0"}, ID: id, Result: result})
}

func (s *Server) sendError(id interface{}, code int, message string) error {
	return s.sendMessage(ResponseMessage{Message: Message{JSONRPC: "2.0"}, ID: id, Error: &ResponseError{Code: code, Message: message}})
}

func (s *Server) sendNotification(method string, params interface{}) error {
	return s.sendMessage(NotificationMessage{Message: Message{JSONRPC: "2.0"}, Method: method, Params: params})
}

// request is a decoded JSON-RPC message.
type request struct {
	ID     interface{}     `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func (s *Server) handleMessage(content json.RawMessage) error {
	var req request
	if err := json.Unmarshal(content, &req); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// The client's answer to a request of the server's own
	// (client/registerCapability): nothing to do.
	if req.Method == "" {
		return nil
	}
	// After shutdown only exit is expected.
	if s.shutdown && req.Method != "exit" {
		if req.ID != nil {
			return s.sendError(req.ID, InvalidRequest, "the server is shutting down")
		}
		return nil
	}

	if !s.initialized && req.Method != "initialize" && req.Method != "exit" {
		if req.ID != nil {
			return s.sendError(req.ID, ServerNotInitialized, "server not initialized")
		}
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(content)
	case "initialized":
		s.initialized = true
		if s.watchFiles {
			// Scripts, declarations and configs changed on disk (a git
			// checkout, a generator) are then reported by the client.
			watchers := []map[string]string{{"globPattern": "**/*.lunar"}, {"globPattern": "**/*.lua"}, {"globPattern": "**/" + project.ConfigName}}
			return s.sendMessage(map[string]any{"jsonrpc": "2.0", "id": "lunar-watch", "method": "client/registerCapability",
				"params": map[string]any{"registrations": []map[string]any{{
					"id": "lunar-watch", "method": "workspace/didChangeWatchedFiles",
					"registerOptions": map[string]any{"watchers": watchers}}}}})
		}
		return nil
	case "shutdown":
		s.shutdown = true
		return s.sendResponse(req.ID, nil)
	case "exit":
		s.exited = true
		return nil
	case "textDocument/didOpen":
		var p DidOpenTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		path := uriToPath(p.TextDocument.URI)
		s.docs[path] = p.TextDocument.Text
		s.clientURIs[path] = p.TextDocument.URI
		s.changed()
		return nil
	case "textDocument/didChange":
		var p DidChangeTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		if n := len(p.ContentChanges); n > 0 {
			// Full sync: the last change is the whole new text.
			s.docs[uriToPath(p.TextDocument.URI)] = p.ContentChanges[n-1].Text
		}
		s.changed()
		return nil
	case "textDocument/didClose":
		var p DidCloseTextDocumentParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return err
		}
		delete(s.docs, uriToPath(p.TextDocument.URI))
		s.changed()
		return nil
	case "workspace/didChangeWatchedFiles":
		s.projectsIn = nil // a config may have come or gone
		s.changed()
		return nil
	case "textDocument/didSave":
		s.changed()
		return nil
	}

	if req.ID == nil {
		return nil // an unhandled notification
	}

	// Answer from the current text, not from before the last keystroke.
	s.flush()

	var result interface{}
	var err error
	switch req.Method {
	case "textDocument/definition":
		result, err = withPosition(req.Params, s.definition)
	case "textDocument/references":
		var p ReferenceParams
		if err = json.Unmarshal(req.Params, &p); err == nil {
			result = s.references(p.TextDocument.URI, p.Position, p.Context.IncludeDeclaration)
		}
	case "textDocument/hover":
		result, err = withPosition(req.Params, s.hover)
	case "textDocument/prepareRename":
		result, err = withPosition(req.Params, s.prepareRename)
	case "textDocument/rename":
		var p RenameParams
		if err = json.Unmarshal(req.Params, &p); err == nil {
			result, err = s.rename(p.TextDocument.URI, p.Position, p.NewName)
		}
	case "textDocument/completion":
		result, err = withPosition(req.Params, s.completion)
	case "textDocument/signatureHelp":
		result, err = withPosition(req.Params, s.signatureHelp)
	case "textDocument/documentSymbol":
		var p DocumentSymbolParams
		if err = json.Unmarshal(req.Params, &p); err == nil {
			result = s.documentSymbols(p.TextDocument.URI)
		}
	case "workspace/symbol":
		var p WorkspaceSymbolParams
		if err = json.Unmarshal(req.Params, &p); err == nil {
			result = s.workspaceSymbols(p.Query)
		}
	default:
		return s.sendError(req.ID, MethodNotFound, "method not supported: "+req.Method)
	}
	if err != nil {
		return s.sendError(req.ID, RequestFailed, err.Error())
	}
	return s.sendResponse(req.ID, result)
}

// withPosition decodes TextDocumentPositionParams for a handler.
func withPosition(raw json.RawMessage, handler func(uri string, pos Position) (interface{}, error)) (interface{}, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return handler(p.TextDocument.URI, p.Position)
}

func (s *Server) handleInitialize(content json.RawMessage) error {
	var req struct {
		ID     interface{}      `json:"id"`
		Params InitializeParams `json:"params"`
	}
	if err := json.Unmarshal(content, &req); err != nil {
		return s.sendError(req.ID, InvalidParams, err.Error())
	}
	switch {
	case len(req.Params.WorkspaceFolders) > 0:
		for _, f := range req.Params.WorkspaceFolders {
			s.roots = append(s.roots, uriToPath(f.URI))
		}
	case req.Params.RootURI != "":
		s.roots = []string{uriToPath(req.Params.RootURI)}
	}
	if len(s.roots) > 0 {
		s.root = s.roots[0]
	}
	s.dirty = true
	if w := req.Params.Capabilities.Workspace.DidChangeWatchedFiles; w != nil && w.DynamicRegistration {
		s.watchFiles = true
	}
	// Requests may follow as soon as the client has this response.
	s.initialized = true

	return s.sendResponse(req.ID, InitializeResult{
		ServerInfo: &ServerInfo{Name: "lunar", Version: version.Version},
		Capabilities: ServerCapabilities{
			TextDocumentSync:        &TextDocumentSyncOptions{OpenClose: true, Change: SyncFull, Save: &SaveOptions{IncludeText: false}},
			HoverProvider:           true,
			CompletionProvider:      &CompletionOptions{TriggerCharacters: []string{".", ":"}},
			SignatureHelpProvider:   &SignatureHelpOptions{TriggerCharacters: []string{"(", ","}},
			DefinitionProvider:      true,
			ReferencesProvider:      true,
			DocumentSymbolProvider:  true,
			WorkspaceSymbolProvider: true,
			RenameProvider:          RenameOptions{PrepareProvider: true},
		},
	})
}

// changed schedules a re-analysis. Typing produces a burst of changes, so
// it waits for a pause; any request in the meantime analyses first.
func (s *Server) changed() {
	s.dirty = true
	if s.debounce <= 0 {
		s.flush()
		return
	}
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(s.debounce, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.flush()
	})
}

// flush re-analyses if anything changed since the last analysis. The
// caller holds s.mu.
func (s *Server) flush() {
	if !s.dirty {
		return
	}
	s.dirty = false
	s.analyze()
}

// analyze checks the workspace and publishes diagnostics for every file,
// clearing those that have none now. The caller holds s.mu.
func (s *Server) analyze() {
	// A compiler bug must not take the editor's server down with it.
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("internal error while analysing: %v\n%s", r, debug.Stack())
			s.sendNotification("window/showMessage", map[string]any{
				"type": 1, "message": fmt.Sprintf("Lunar: internal compiler error: %v", r)})
		}
	}()

	overlay := map[string]string{}
	for path, text := range s.docs {
		overlay[path] = text
	}
	s.files = map[string]*source.File{}
	s.progs = nil
	s.broken = map[string]bool{}

	// Each file belongs to the lunar.config.json found from its own folder,
	// as for `lunar build` run there; "" when there is none.
	configs := map[string]string{}
	configOf := func(path string) string {
		dir := filepath.Dir(path)
		if cfg, ok := configs[dir]; ok {
			return cfg
		}
		cfg, _ := project.FindConfig(dir)
		configs[dir] = cfg
		return cfg
	}

	// One analysis per config: the workspace's own (or the workspace folder
	// itself when it has none), and one for each open document's.
	type run struct {
		key  string // the config that owns what this run reports
		opts project.Options
	}
	// Each analysis starts from the previous one of the same project, so an
	// edit re-checks only the edited file and what imports it.
	prevProgs := s.lastProgs
	s.lastProgs = map[string]*project.Program{}
	cacheKey := func(r run) string {
		if len(r.opts.Files) > 0 {
			return "files:" + r.opts.Dir
		}
		return "project:" + r.key + "\x00" + r.opts.Dir
	}
	var runs []run
	seen := map[string]bool{}
	// Each workspace folder: the project it is in, if any, and every
	// project inside it, checked whether or not one of its files is open.
	// A folder with no config (a repository's root) is not one program:
	// its loose files are checked one folder at a time, below.
	for _, root := range s.roots {
		var cfgs []string
		if cfg, found := project.FindConfig(root); found {
			cfgs = append(cfgs, cfg)
		}
		cfgs = append(cfgs, s.projectsUnder(root)...)
		for _, cfg := range cfgs {
			if !seen[cfg] {
				runs = append(runs, run{cfg, project.Options{ConfigPath: cfg, Overlay: overlay}})
				seen[cfg] = true
			}
		}
	}
	docs := make([]string, 0, len(s.docs))
	for path := range s.docs {
		docs = append(docs, path)
	}
	sort.Strings(docs)
	for _, path := range docs {
		if cfg := configOf(path); cfg != "" && !seen[cfg] {
			runs = append(runs, run{cfg, project.Options{ConfigPath: cfg, Overlay: overlay}})
			seen[cfg] = true
		}
	}

	var diags diag.List
	covered := map[string]bool{}
	fresh := map[string]*project.Module{}
	analyse := func(r run, owns func(path string) bool) {
		p, openDiags := project.Open(r.opts)
		if openDiags.HasErrors() {
			if r.key == "" && len(r.opts.Files) > 0 {
				// A file on its own (outside its project's rootDir, say):
				// the problem is shown on the file.
				for _, f := range r.opts.Files {
					for _, d := range openDiags {
						d.File, d.Line, d.Column, d.EndLine, d.EndColumn = f, 1, 1, 1, 1
						diags = append(diags, d)
					}
				}
				return
			}
			diags = append(diags, s.configProblem(r.key, openDiags)...)
			return
		}
		delete(s.configErrors, r.key)
		prog, checkDiags := p.CheckReusing(prevProgs[cacheKey(r)])
		s.lastProgs[cacheKey(r)] = prog
		s.progs = append(s.progs, prog)
		for _, m := range prog.Modules {
			if !owns(m.Path) {
				continue
			}
			covered[m.Path] = true
			if m.Broken {
				s.broken[m.Path] = true
				// A module that does not parse keeps its last good index.
				if old := s.modules[m.Path]; old != nil {
					fresh[m.Path] = old
				}
			} else if m.Index != nil {
				fresh[m.Path] = m
			}
		}
		for _, d := range checkDiags {
			if d.File == "" || strings.HasPrefix(d.File, "<") || owns(d.File) {
				diags = append(diags, d)
			}
		}
	}
	for _, r := range runs {
		key := r.key
		analyse(r, func(path string) bool { return configOf(path) == key })
	}

	// Open documents no project covers (no config, or outside its
	// rootDir) are analysed on their own, a folder at a time.
	byDir := map[string][]string{}
	var dirs []string
	for _, path := range docs {
		if covered[path] {
			continue
		}
		dir := filepath.Dir(path)
		if byDir[dir] == nil {
			dirs = append(dirs, dir)
		}
		byDir[dir] = append(byDir[dir], path)
	}
	for _, dir := range dirs {
		files := byDir[dir]
		own := map[string]bool{}
		for _, f := range files {
			own[f] = true
		}
		analyse(run{"", project.Options{Dir: dir, Files: files, Overlay: overlay}}, func(path string) bool { return own[path] })
	}

	// Only the modules some project still has: a deleted or moved file
	// leaves the index (rename used to edit it, recreating it).
	s.modules = fresh
	s.publish(diags)
}

// configProblem places the errors from opening a project on its config
// file, and shows the first in a message once, so a typo in the config
// does not just make every diagnostic disappear.
func (s *Server) configProblem(configPath string, diags diag.List) diag.List {
	out := make(diag.List, 0, len(diags))
	for _, d := range diags {
		if d.File == "" && configPath != "" {
			d.File, d.Line, d.Column, d.EndLine, d.EndColumn = configPath, 1, 1, 1, 1
		}
		out = append(out, d)
	}
	msg := diags[0].Message
	s.logger.Printf("cannot open the project: %s", msg)
	if s.configErrors[configPath] != msg {
		s.configErrors[configPath] = msg
		s.sendNotification("window/showMessage", map[string]any{"type": 1, "message": "Lunar: " + msg})
	}
	return out
}

// publish sends the diagnostics for every file that has some, and an empty
// list for every file that had some before.
func (s *Server) publish(diags diag.List) {
	byFile := map[string][]Diagnostic{}
	for _, d := range diags {
		if d.File == "" || strings.HasPrefix(d.File, "<") {
			continue
		}
		byFile[d.File] = append(byFile[d.File], Diagnostic{
			Range:    s.rangeOf(d.File, d.Line, d.Column, d.EndLine, d.EndColumn),
			Severity: diagnosticSeverity(d.Severity),
			Code:     d.Code,
			Source:   "lunar",
			Message:  d.Message,
		})
	}
	for file := range s.published {
		if _, still := byFile[file]; !still {
			s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{URI: s.uriOf(file), Diagnostics: []Diagnostic{}})
			delete(s.published, file)
		}
	}
	for file, list := range byFile {
		s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{URI: s.uriOf(file), Diagnostics: list})
		s.published[file] = true
	}
}

func diagnosticSeverity(sev diag.Severity) DiagnosticSeverity {
	switch sev {
	case diag.Warning:
		return SeverityWarning
	case diag.Info:
		return SeverityInformation
	}
	return SeverityError
}

// uriOf names a file to the editor: in the editor's own spelling if it
// opened it, otherwise built from the path.
func (s *Server) uriOf(path string) string {
	if uri, ok := s.clientURIs[path]; ok {
		return uri
	}
	return pathToURI(path)
}

// projectsUnder lists the lunar.config.json files in root's tree, skipping
// hidden folders and node_modules, a few levels deep. The list is kept
// until a watched-file event.
func (s *Server) projectsUnder(root string) []string {
	if cfgs, ok := s.projectsIn[root]; ok {
		return cfgs
	}
	var cfgs []string
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			switch {
			case e.IsDir() && depth < 6 && !strings.HasPrefix(e.Name(), ".") && e.Name() != "node_modules":
				walk(filepath.Join(dir, e.Name()), depth+1)
			case e.Name() == project.ConfigName && !e.IsDir():
				cfgs = append(cfgs, filepath.Join(dir, e.Name()))
			}
		}
	}
	walk(root, 0)
	sort.Strings(cfgs)
	if s.projectsIn == nil {
		s.projectsIn = map[string][]string{}
	}
	s.projectsIn[root] = cfgs
	return cfgs
}
