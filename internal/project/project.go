// Package project is the one compile pipeline the CLI and the language
// server share: it finds the configuration, the scripts and the
// declarations, resolves imports into a module graph, checks every module,
// and writes the Lua.
package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"lunar/internal/diag"
)

// ConfigName is the configuration file lunar looks for.
const ConfigName = "lunar.config.json"

// Config is lunar.config.json. Paths are relative to the file.
type Config struct {
	// RootDir is the scripts folder; imports and require names are relative
	// to it. Defaults to the folder holding the config.
	RootDir string `json:"rootDir,omitempty"`
	// OutDir receives the Lua, mirroring RootDir's layout.
	OutDir string `json:"outDir,omitempty"`
	// Target is "lua53" (the default) or "lua54".
	Target string `json:"target,omitempty"`
	// Declarations are extra *.d.lunar files or folders, such as the
	// engine's API, anywhere on disk.
	Declarations []string `json:"declarations,omitempty"`
	// CopyLua copies plain .lua scripts under RootDir to OutDir. Default true.
	CopyLua *bool `json:"copyLua,omitempty"`
	// SourceMap writes <file>.lua.map next to each output.
	SourceMap bool `json:"sourceMap,omitempty"`
	// DataFiles are globs, relative to RootDir, of scripts the engine loads
	// with no libraries opened; they may not use any global.
	DataFiles []string `json:"dataFiles,omitempty"`
}

// LoadConfig reads a config file. Unknown keys are errors, so a misspelt
// option fails instead of being ignored.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Notepad saves UTF-8 with a byte-order mark.
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	// encoding/json matches keys without regard to case, which would let
	// "rootdir" stand for "rootDir"; check the spelling exactly first.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	known := map[string]bool{}
	t := reflect.TypeOf(Config{})
	for i := 0; i < t.NumField(); i++ {
		known[strings.Split(t.Field(i).Tag.Get("json"), ",")[0]] = true
	}
	for key := range raw {
		if !known[key] {
			return nil, fmt.Errorf("%s: unknown option %q", path, key)
		}
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

// FindConfig walks up from dir looking for lunar.config.json.
func FindConfig(dir string) (string, bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, ConfigName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Options say what to compile. Zero values defer to the config file.
type Options struct {
	// Dir is where the search for lunar.config.json starts; defaults to
	// the working directory, or to the first file's folder with Files.
	Dir string
	// ConfigPath names the config file instead of searching for one.
	ConfigPath string
	// OutDir overrides the config's outDir.
	OutDir string
	// Target overrides the config's target.
	Target string
	// Decls adds declaration files or folders.
	Decls []string
	// SourceMap, when set, overrides the config's sourceMap.
	SourceMap *bool
	// Files compiles only these files; the modules they import are
	// checked but not emitted. Empty means every script under the root.
	Files []string
	// Overlay holds unsaved editor buffers by absolute path.
	Overlay map[string]string
	// RequireConfig refuses to go on without a lunar.config.json (or an
	// OutDir): lunar build compiles a project, not whatever folder it is
	// run in.
	RequireConfig bool
}

// Project is what Open found: where everything is and how to compile it.
type Project struct {
	Root      string // absolute scripts folder
	OutDir    string // absolute output folder; "" if none was given
	Target    string
	SourceMap bool
	CopyLua   bool
	DataFiles []string

	DeclFiles []string // absolute *.d.lunar paths
	Sources   []string // absolute *.lunar paths to emit
	LuaFiles  []string // absolute *.lua paths to copy through

	overlay map[string]string
	// configPath is the config file, when there is one; wholeFolder says
	// the project is the whole root, not files named on the command line.
	configPath  string
	wholeFolder bool
}

var supportedTargets = map[string]bool{"lua53": true, "lua54": true}

// configError is a diagnostic about the configuration, not about a script.
// ioError is a failure to read the project's files: exit 3, not a
// mistake in the config.
func ioError(err error) diag.List {
	return diag.List{{Severity: diag.Error, Code: "io", Message: err.Error()}}
}

// ConfigFile is the project's lunar.config.json, or "" without one.
func (p *Project) ConfigFile() string { return p.configPath }

// sameFolder says whether two paths name one folder, through symlinks too.
func sameFolder(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ia, ib)
}

// within says whether path is inside folder (folder itself excluded).
func within(path, folder string) bool {
	rel, err := filepath.Rel(folder, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func configError(format string, args ...any) diag.List {
	return diag.List{{Severity: diag.Error, Code: "config", Message: fmt.Sprintf(format, args...)}}
}

// Open reads the configuration and finds the scripts and declarations.
func Open(opts Options) (*Project, diag.List) {
	start := opts.Dir
	if start == "" && len(opts.Files) > 0 {
		start = filepath.Dir(opts.Files[0])
	}
	if start == "" {
		start = "."
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return nil, configError("%v", err)
	}

	cfg := &Config{}
	configDir := start
	configPath, found := opts.ConfigPath, opts.ConfigPath != ""
	if !found {
		configPath, found = FindConfig(start)
	}
	if !found && opts.RequireConfig && opts.OutDir == "" {
		return nil, configError("no %s in %s or a folder above it: lunar build needs one, giving rootDir and outDir (or pass --out)", ConfigName, start)
	}
	if found {
		if cfg, err = LoadConfig(configPath); err != nil {
			return nil, configError("%v", err)
		}
		if configDir, err = filepath.Abs(filepath.Dir(configPath)); err != nil {
			return nil, configError("%v", err)
		}
	}

	// A path in the config is relative to its folder, unless absolute.
	fromConfig := func(path string) string {
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return filepath.Join(configDir, path)
	}
	p := &Project{overlay: opts.Overlay, CopyLua: true, SourceMap: cfg.SourceMap, DataFiles: cfg.DataFiles}
	p.Root = fromConfig(cfg.RootDir)
	if found {
		p.configPath, _ = filepath.Abs(configPath)
	}
	p.wholeFolder = len(opts.Files) == 0
	if cfg.CopyLua != nil {
		p.CopyLua = *cfg.CopyLua
	}
	if opts.SourceMap != nil {
		p.SourceMap = *opts.SourceMap
	}

	switch {
	case opts.OutDir != "":
		p.OutDir, _ = filepath.Abs(opts.OutDir)
	case cfg.OutDir != "":
		p.OutDir = fromConfig(cfg.OutDir)
	}

	p.Target = "lua53"
	if cfg.Target != "" {
		p.Target = cfg.Target
	}
	if opts.Target != "" {
		p.Target = opts.Target
	}
	if !supportedTargets[p.Target] {
		return nil, configError("unsupported target '%s' (supported: lua53, lua54)", p.Target)
	}

	if info, err := os.Stat(p.Root); err != nil || !info.IsDir() {
		return nil, configError("rootDir '%s' is not a folder", p.Root)
	}
	// The next build would read the Lua it wrote back in as hand-written
	// scripts. (Single-file compiles write next to the source instead.)
	if p.OutDir != "" && len(opts.Files) == 0 && sameFolder(p.OutDir, p.Root) {
		return nil, configError("outDir is the same folder as rootDir (%s); put the Lua in a folder of its own, such as \"../build\"", p.Root)
	}
	// Outputs mirror the scripts' layout under outDir: one holding rootDir
	// puts them among the scripts (src/src/util.lunar wrote src/util.lua).
	if p.OutDir != "" && len(opts.Files) == 0 && within(p.Root, p.OutDir) {
		return nil, configError("outDir (%s) holds rootDir (%s), so outputs would land among the scripts; put the Lua in a folder beside it, such as \"../build\"", p.OutDir, p.Root)
	}

	// Declarations: everything under the root, plus the configured and
	// flagged paths, which may be anywhere.
	declPaths := []string{p.Root}
	for _, d := range cfg.Declarations {
		declPaths = append(declPaths, fromConfig(d))
	}
	for _, d := range opts.Decls {
		abs, _ := filepath.Abs(d)
		declPaths = append(declPaths, abs)
	}
	seen := map[string]bool{}
	for i, path := range declPaths {
		var files []string
		var err error
		if i == 0 && !found && len(opts.Files) > 0 {
			// No project: a lone file sees the declarations beside it,
			// not everything below its folder.
			files, err = filepath.Glob(filepath.Join(path, "*.d.lunar"))
			sort.Strings(files)
		} else {
			files, err = p.scan(path, ".d.lunar")
		}
		if err != nil && i > 0 {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, configError("declaration path '%s' does not exist", path)
			}
			return nil, ioError(err)
		}
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				p.DeclFiles = append(p.DeclFiles, f)
			}
		}
	}

	if len(opts.Files) > 0 {
		for _, f := range opts.Files {
			abs, err := filepath.Abs(f)
			if err != nil {
				return nil, configError("%v", err)
			}
			if rel, err := filepath.Rel(p.Root, abs); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return nil, configError("%s is outside rootDir (%s) of %s; move it under rootDir, or compile the project with lunar build",
					filepath.Base(abs), p.Root, configPath)
			}
			p.Sources = append(p.Sources, abs)
		}
		return p, nil
	}

	all, err := p.scan(p.Root, ".lunar")
	if err != nil {
		return nil, ioError(err)
	}
	for _, f := range all {
		if !strings.HasSuffix(f, ".d.lunar") {
			p.Sources = append(p.Sources, f)
		}
	}
	if p.CopyLua {
		if p.LuaFiles, err = p.scan(p.Root, ".lua"); err != nil {
			return nil, ioError(err)
		}
	}
	return p, nil
}

// scan lists the files under path (a file or a folder) ending in suffix,
// sorted, skipping hidden folders and the output folder. Folders reached
// through symlinks are scanned like any other (the root included), each
// once, so a link back up the tree does not loop.
func (p *Project) scan(path, suffix string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.HasSuffix(path, suffix) {
			return []string{path}, nil
		}
		return nil, nil
	}
	outInfo, outErr := os.Stat(p.OutDir)
	isOutDir := func(dir string, info fs.FileInfo) bool {
		return p.OutDir != "" && (dir == p.OutDir || (outErr == nil && os.SameFile(info, outInfo)))
	}
	var out []string
	var visited []fs.FileInfo
	var walk func(dir string, info fs.FileInfo) error
	walk = func(dir string, info fs.FileInfo) error {
		for _, v := range visited {
			if os.SameFile(v, info) {
				return nil
			}
		}
		visited = append(visited, info)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			file := filepath.Join(dir, e.Name())
			st, err := os.Stat(file) // through a symlink
			if err != nil {
				continue // a dangling link
			}
			if st.IsDir() {
				if strings.HasPrefix(e.Name(), ".") || isOutDir(file, st) {
					continue
				}
				if err := walk(file, st); err != nil {
					return err
				}
				continue
			}
			if strings.HasSuffix(file, suffix) && (suffix != ".lunar" || !strings.HasSuffix(file, ".lunar-tmp")) {
				out = append(out, file)
			}
		}
		return nil
	}
	err = walk(path, info)
	sort.Strings(out)
	return out, err
}

// read returns a file's text, preferring an unsaved buffer.
func (p *Project) read(path string) (string, error) {
	if text, ok := p.overlay[path]; ok {
		return text, nil
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// rel is path relative to the root, with forward slashes.
func (p *Project) rel(path string) string {
	r, err := filepath.Rel(p.Root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}
