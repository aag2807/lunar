package project

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

// checkText runs the whole pipeline on one file's text, as the language
// server does on every keystroke, and reports a panic or a hang.
func checkText(dir, text string) (err error) {
	path := filepath.Join(dir, "main.lunar")
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic: %v\n%s", r, lunarFrames(debug.Stack()))
			}
		}()
		p, diags := Open(Options{Dir: dir, Files: []string{path}, Overlay: map[string]string{path: text}})
		if diags.HasErrors() {
			done <- fmt.Errorf("open: %s", diags[0].Message)
			return
		}
		prog, _ := p.Check()
		prog.Emit()
		done <- nil
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("hang: no result after 5s")
	}
}

// lunarFrames keeps the stack lines that point into lunar's own code.
func lunarFrames(stack []byte) string {
	var out []string
	for _, line := range strings.Split(string(stack), "\n") {
		if strings.Contains(line, "lunar/internal/") && !strings.Contains(line, "robust_test") {
			out = append(out, strings.TrimSpace(line))
		}
	}
	if len(out) > 8 {
		out = out[:8]
	}
	return strings.Join(out, "\n")
}

// TestHalfWrittenFilesNeitherPanicNorHang feeds the checker every example
// cut off at the end of each line and halfway through it: the states a
// buffer passes through while someone types it.
func TestHalfWrittenFilesNeitherPanicNorHang(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "examples", "features", "*.lunar"))
	data, _ := filepath.Glob(filepath.Join("..", "..", "examples", "data", "*.lunar"))
	files = append(files, data...)
	if len(files) == 0 {
		t.Fatal("no examples found")
	}
	dir := t.TempDir()
	for _, file := range files {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		var cuts []int
		start := 0
		for i := 0; i <= len(text); i++ {
			if i == len(text) || text[i] == '\n' {
				cuts = append(cuts, start+(i-start)/2, i)
				start = i + 1
			}
		}
		for _, cut := range cuts {
			if err := checkText(dir, text[:cut]); err != nil {
				t.Fatalf("%s cut at byte %d: %v\n--- text ---\n%s", filepath.Base(file), cut, err, text[:cut])
			}
		}
	}
}

// TestUnfinishedStatementsNeitherPanicNorHang inserts the start of each
// kind of statement into a valid program, at the end and in the middle.
func TestUnfinishedStatementsNeitherPanicNorHang(t *testing.T) {
	starts := []string{
		"function", "local function", "function f(", "function f(x: number", "function g<T>(x: T",
		"if x", "if x then", "while", "for", "for i = 1, do", "for k, v in", "repeat", "do",
		"class", "class A extends", "abstract class", "interface", "interface I extends", "enum",
		"type", "type T =", "type T = string |", "namespace", "export", "export function", "export class",
		"export const", "export local", "import", "import {", "import { a }", "import { a } from",
		"import * as", "declare", "declare function", "declare class", "declare namespace", "@",
		"match", "match x with", "local x =", "local x: = 1", "local { a, b } = t", "return",
		"local s = \"abc", "local s = 'abc", "local s = `abc", "local s = `abc ${", "local s = [[abc",
		"--[[ comment", "local t = {", "local t = { a = 1,", "x.", "x:", "x.y(", "local f = function(",
		"local f = (x) =>", "local a: number[", "local v = a ?? ", "local v = a?.", "local n = 0x",
		"local n = 1e", "get", "set", "local x = 1 +", "goto", "::label",
		"async", "async function", "async function f(", "local async", "local async function",
		"await", "await f(", "local x = async (", "local x = async (x", "declare async",
		"local m: { x: number } = { x = 1 }", "local t: [() => void] = { a }",
		"local f: (self: any) => void", "x: number = 1",
	}
	program := "local a = 1\nlocal b = 2\nprint(a + b)\n"
	dir := t.TempDir()
	for _, start := range starts {
		for _, text := range []string{program + start, program + start + "\n" + program, start} {
			if err := checkText(dir, text); err != nil {
				t.Errorf("%q: %v", strings.ReplaceAll(text, "\n", `\n`), err)
			}
		}
	}
}

// A declaration file being typed passes through the same half-written
// states; checking it must not panic either.
func TestHalfWrittenDeclarationsNeitherPanicNorHang(t *testing.T) {
	for _, decl := range []string{
		"declare const", "declare local", "declare interface", "declare enum", "declare type",
		"declare function", "declare class", "declare namespace", "declare const x:", "declare type T =",
		"declare interface I\n    x:", "declare class C\n    get", "declare enum E\n    A =",
	} {
		dir := t.TempDir()
		api := filepath.Join(dir, "api.d.lunar")
		os.WriteFile(api, []byte(decl+"\n"), 0o644)
		main := filepath.Join(dir, "main.lunar")
		done := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("panic: %v\n%s", r, lunarFrames(debug.Stack()))
				}
			}()
			p, diags := Open(Options{Dir: dir, Files: []string{main}, Overlay: map[string]string{main: "print(1)\n"}})
			if diags.HasErrors() {
				done <- fmt.Errorf("open: %s", diags[0].Message)
				return
			}
			p.Check()
			done <- nil
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("%q: %v", decl, err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("%q: hang", decl)
		}
	}
}
