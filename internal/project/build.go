package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	if !p.SourceMap {
		return codegen.GenerateWithTarget(m.Statements, p.Target), "", nil
	}
	lua, sm := codegen.GenerateWithSourceMapAndTarget(m.Statements, m.Path, rel, p.Target)
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
		// Another project's outputs would all look stale to this one.
		if data, err := os.ReadFile(filepath.Join(p.OutDir, manifestName)); err == nil {
			var m manifest
			if json.Unmarshal(data, &m) == nil && m.Root != "" && filepath.Clean(m.Root) != filepath.Clean(p.Root) {
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

func writeOutputs(outDir, root string, outs []Output) error {
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
	// Move each old file aside and the new one into its place. If any step
	// fails, undo every step taken, so the folder keeps the last good
	// build rather than half of each.
	type step struct {
		target, backup string
		hadOld, placed bool
	}
	var done []step
	undo := func() {
		for i := len(done) - 1; i >= 0; i-- {
			st := done[i]
			if st.placed {
				os.Remove(st.target)
			}
			if st.hadOld {
				renameFile(st.backup, st.target)
			}
		}
		cleanup()
	}
	for i, o := range outs {
		st := step{target: filepath.Join(outDir, filepath.FromSlash(o.Rel))}
		st.backup = st.target + ".lunar-old"
		if info, err := os.Lstat(st.target); err == nil && !info.IsDir() {
			if err := retry(func() error { return renameFile(st.target, st.backup) }); err != nil {
				undo()
				return fmt.Errorf("replacing %s: %w (nothing was changed)", st.target, err)
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
	current := map[string]bool{}
	for _, o := range outs {
		current[o.Rel] = true
	}
	manifestPath := filepath.Join(outDir, manifestName)
	if data, err := os.ReadFile(manifestPath); err == nil {
		var old manifest
		if json.Unmarshal(data, &old) == nil {
			for _, rel := range old.Files {
				if !current[rel] && !strings.Contains(rel, "..") {
					os.Remove(filepath.Join(outDir, filepath.FromSlash(rel)))
				}
			}
		}
	}
	m := manifest{Root: root, Files: make([]string, 0, len(outs))}
	for _, o := range outs {
		m.Files = append(m.Files, o.Rel)
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(manifestPath, append(data, '\n'), 0o644)
}
