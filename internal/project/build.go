package project

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"lunar/internal/codegen"
	"lunar/internal/diag"
)

// Output is one file a build writes, relative to the output folder.
type Output struct {
	Rel     string // slash-separated, e.g. "ai/brain.lua"
	Content []byte
	// Source is the .lunar or .lua file it came from.
	Source string
}

// Emit generates the Lua for every module that is emitted, plus the plain
// .lua scripts copied through. It reports a .lua script that would collide
// with a compiled one.
func (prog *Program) Emit() ([]Output, diag.List) {
	p := prog.Project
	var outs []Output
	var diags diag.List
	written := map[string]string{} // output rel -> source

	for _, m := range prog.Modules {
		if !m.Emit || m.Broken {
			continue
		}
		rel := strings.TrimSuffix(m.Rel, ".lunar") + ".lua"
		code, mapJSON, err := p.generate(m, rel)
		if err != nil {
			diags = append(diags, diag.Diagnostic{File: m.Path, Severity: diag.Error, Code: "internal", Message: err.Error()})
			continue
		}
		if mapJSON != "" {
			outs = append(outs, Output{Rel: rel + ".map", Content: []byte(mapJSON), Source: m.Path})
		}
		outs = append(outs, Output{Rel: rel, Content: []byte(code), Source: m.Path})
		written[rel] = m.Path
	}

	for _, path := range p.LuaFiles {
		rel := p.rel(path)
		if other, clash := written[rel]; clash {
			diags = append(diags, diag.Diagnostic{File: path, Severity: diag.Error, Code: "output-conflict",
				Message: fmt.Sprintf("%s and %s would both write %s; rename or remove one", p.rel(path), p.rel(other), rel)})
			continue
		}
		data, err := p.read(path)
		if err != nil {
			diags = append(diags, ioDiag(path, err))
			continue
		}
		outs = append(outs, Output{Rel: rel, Content: []byte(data), Source: path})
	}

	sort.Slice(outs, func(i, j int) bool { return outs[i].Rel < outs[j].Rel })
	return outs, diags
}

// generate turns one module into Lua (and its source map, if enabled). A
// code generator panic becomes an error on the module.
func (p *Project) generate(m *Module, rel string) (code, mapJSON string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal compiler error generating Lua: %v", r)
		}
	}()
	// A field decorator's info names the script as the host knows it,
	// relative to rootDir: enemies/guard.lunar.
	if !p.SourceMap {
		g := codegen.NewWithTarget(p.Target)
		g.ModuleName = p.rel(m.Path)
		return g.Generate(m.Statements), "", nil
	}
	g := codegen.NewWithSourceMapAndTarget(m.Path, rel, p.Target)
	g.ModuleName = p.rel(m.Path)
	lua := g.Generate(m.Statements)
	sm := g.GetSourceMap()
	if mapJSON, err = sm.ToJSON(); err != nil {
		return "", "", err
	}
	return lua + "\n" + sm.GenerateComment(filepath.Base(rel)+".map") + "\n", mapJSON, nil
}

// Result describes a build.
type Result struct {
	Files   int  // .lunar modules compiled
	Written bool // whether the output folder was updated
}

// manifestName records, inside the output folder, which files the last
// build wrote, so the next one can remove only its own stale outputs.
const manifestName = ".lunar-manifest.json"

type manifest struct {
	// Root is the rootDir of the project that wrote the folder.
	Root  string   `json:"root,omitempty"`
	Files []string `json:"files"`
}

// Build checks the whole project and, only if it has no errors, writes
// every output. Each file is staged next to its target and renamed over
// it, and a failed build writes nothing at all, so the last good Lua stays
// in place.
func Build(p *Project) (Result, diag.List) {
	if p.OutDir != "" {
		// Two builds writing one folder at once removed each other's
		// staged files and backups; the second waits for the first.
		unlock, err := lockOutDir(p.OutDir)
		if err != nil {
			return Result{}, diag.List{{Severity: diag.Error, Code: "io", Message: err.Error()}}
		}
		defer unlock()
		// Another project's outputs would all look stale to this one.
		if data, err := os.ReadFile(filepath.Join(p.OutDir, manifestName)); err == nil {
			var m manifest
			if json.Unmarshal(data, &m) == nil && m.Root != "" && !sameDir(m.Root, p.Root) {
				return Result{}, configError("outDir %s holds the build of another project (rootDir %s); give each project its own outDir", p.OutDir, m.Root)
			}
		}
		recoverLeftovers(p.OutDir)
	}
	prog, diags := p.Check()
	res := Result{}
	for _, m := range prog.Modules {
		if m.Emit {
			res.Files++
		}
	}
	if diags.HasErrors() {
		return res, diags
	}

	outs, emitDiags := prog.Emit()
	diags = append(diags, emitDiags...)
	if diags.HasErrors() {
		return res, diags
	}
	if p.OutDir == "" {
		return res, configError("no output folder: set outDir in %s or pass --out", ConfigName)
	}

	if err := writeOutputs(p.OutDir, p.Root, outs); err != nil {
		return res, append(diags, diag.Diagnostic{Severity: diag.Error, Code: "io", Message: err.Error()})
	}
	res.Written = true
	return res, diags
}

// sameDir says whether two paths name one folder, however each is
// spelled: through a symlink, or with another drive-letter case.
func sameDir(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ia, ib)
}

// lockPath is the lock file for an output folder: in the temp folder, so
// a build that writes nothing leaves outDir untouched, and named after the
// folder itself, so every spelling of it shares one lock.
func lockPath(outDir string) string {
	key, _ := filepath.Abs(outDir)
	if real, err := filepath.EvalSymlinks(key); err == nil {
		key = real
	} else if parent, err := filepath.EvalSymlinks(filepath.Dir(key)); err == nil {
		key = filepath.Join(parent, filepath.Base(key))
	}
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(os.TempDir(), fmt.Sprintf("lunar-build-%x.lock", sum[:8]))
}

// lockWait is how long a build waits for another one writing its folder.
var lockWait = 2 * time.Minute

// lockOutDir waits until no other build holds outDir's lock, takes it, and
// returns its release. The operating system drops the lock of a build that
// dies, so a killed build never leaves the folder locked.
func lockOutDir(outDir string) (func(), error) {
	path := lockPath(outDir)
	deadline := time.Now().Add(lockWait)
	for {
		release, busy, err := tryLock(path)
		if err != nil {
			return nil, fmt.Errorf("locking %s: %w", outDir, err)
		}
		if !busy {
			return release, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("another lunar build is writing %s (waited %v)", outDir, lockWait)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// retry runs op up to three times: on Windows a rename fails for a moment
// while another program (an antivirus, the editor) has the file open.
func retry(op func() error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = op(); err == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}
	return err
}

// renameFile is os.Rename; tests replace it to make one rename fail.
var renameFile = os.Rename

// recoverLeftovers repairs what a build killed while writing left behind:
// an old file it had moved aside goes back if nothing replaced it, and
// staged copies are removed.
func recoverLeftovers(outDir string) {
	filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch {
		case strings.HasSuffix(path, ".lunar-tmp"):
			os.Remove(path)
		case strings.HasSuffix(path, ".lunar-old"):
			target := strings.TrimSuffix(path, ".lunar-old")
			if _, err := os.Lstat(target); errors.Is(err, fs.ErrNotExist) {
				renameFile(path, target)
			} else {
				os.Remove(path)
			}
		}
		return nil
	})
}

// linkFile is os.Link; tests replace it to see the fallback.
var linkFile = os.Link

func writeOutputs(outDir, root string, outs []Output) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", outDir, err)
	}
	manifestPath := filepath.Join(outDir, manifestName)
	var old manifest
	if data, err := os.ReadFile(manifestPath); err == nil {
		json.Unmarshal(data, &old)
	}
	// The manifest is written with the outputs and undone with them: one
	// that failed to update after the outputs did made a build that said
	// it wrote nothing, and left outputs no build would remove.
	m := manifest{Root: root, Files: make([]string, 0, len(outs))}
	for _, o := range outs {
		m.Files = append(m.Files, o.Rel)
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	outs = append(outs[:len(outs):len(outs)], Output{Rel: manifestName, Content: append(data, '\n')})

	var staged []string
	cleanup := func() {
		for _, s := range staged {
			os.Remove(s)
		}
	}
	for _, o := range outs {
		target := filepath.Join(outDir, filepath.FromSlash(o.Rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			cleanup()
			return fmt.Errorf("creating %s: %w", filepath.Dir(target), err)
		}
		tmp := target + ".lunar-tmp"
		if err := os.WriteFile(tmp, o.Content, 0o644); err != nil {
			cleanup()
			return fmt.Errorf("writing %s: %w", tmp, err)
		}
		staged = append(staged, tmp)
	}
	// Keep each old file under a second name, then rename the new one over
	// it: a reader never finds the file missing. Where a second name
	// cannot be made (no hard links), the old file is moved aside instead.
	// If any step fails, undo every step taken, so the folder keeps the
	// last good build rather than half of each.
	type step struct {
		target, backup string
		hadOld, placed bool
	}
	var done []step
	undo := func() {
		for i := len(done) - 1; i >= 0; i-- {
			st := done[i]
			if st.hadOld {
				renameFile(st.backup, st.target)
				os.Remove(st.backup) // when both names were one file
			} else if st.placed {
				os.Remove(st.target)
			}
		}
		cleanup()
	}
	for i, o := range outs {
		st := step{target: filepath.Join(outDir, filepath.FromSlash(o.Rel))}
		st.backup = st.target + ".lunar-old"
		if info, err := os.Lstat(st.target); err == nil && !info.IsDir() {
			os.Remove(st.backup)
			if linkFile(st.target, st.backup) != nil {
				if err := retry(func() error { return renameFile(st.target, st.backup) }); err != nil {
					undo()
					return fmt.Errorf("replacing %s: %w (nothing was changed)", st.target, err)
				}
			}
			st.hadOld = true
		}
		done = append(done, st)
		if err := retry(func() error { return renameFile(staged[i], st.target) }); err != nil {
			undo()
			return fmt.Errorf("replacing %s: %w (nothing was changed)", st.target, err)
		}
		done[len(done)-1].placed = true
	}
	for _, st := range done {
		if st.hadOld {
			os.Remove(st.backup)
		}
	}

	// Remove what the previous build wrote and this one did not.
	current := map[string]bool{manifestName: true}
	for _, rel := range m.Files {
		current[rel] = true
	}
	for _, rel := range old.Files {
		if !current[rel] && !strings.Contains(rel, "..") {
			os.Remove(filepath.Join(outDir, filepath.FromSlash(rel)))
		}
	}
	return nil
}
