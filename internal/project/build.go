package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	Files []string `json:"files"`
}

// Build checks the whole project and, only if it has no errors, writes
// every output. Each file is staged next to its target and renamed over
// it, and a failed build writes nothing at all, so the last good Lua stays
// in place.
func Build(p *Project) (Result, diag.List) {
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

	if err := writeOutputs(p.OutDir, outs); err != nil {
		return res, append(diags, diag.Diagnostic{Severity: diag.Error, Code: "io", Message: err.Error()})
	}
	res.Written = true
	return res, diags
}

func writeOutputs(outDir string, outs []Output) error {
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
	for i, o := range outs {
		target := filepath.Join(outDir, filepath.FromSlash(o.Rel))
		if err := os.Rename(staged[i], target); err != nil {
			cleanup()
			return fmt.Errorf("replacing %s: %w", target, err)
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
	m := manifest{Files: make([]string, 0, len(outs))}
	for _, o := range outs {
		m.Files = append(m.Files, o.Rel)
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(manifestPath, append(data, '\n'), 0o644)
}
