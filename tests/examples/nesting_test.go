package examples

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// nestings are code shapes nested n deep. A level of the first kind costs
// Lua one of its 200 nesting levels, of the second kind two.
var nestings = map[string]func(n int) string{
	"do": func(n int) string { return strings.Repeat("do\n", n) + "print(1)\n" + strings.Repeat("end\n", n) },
	"if": func(n int) string {
		return "local x = 1\n" + strings.Repeat("if x then\n", n) + "print(1)\n" + strings.Repeat("end\n", n)
	},
	"table": func(n int) string {
		return "local t = " + strings.Repeat("{ ", n) + "1" + strings.Repeat(" }", n) + "\nprint(type(t))\n"
	},
	"paren": func(n int) string {
		return "local v = " + strings.Repeat("(", n) + "1" + strings.Repeat(")", n) + "\nprint(v)\n"
	},
	"concat": func(n int) string { return "local v = " + strings.Repeat(`"a" .. `, n-1) + `"a"` + "\nprint(#v)\n" },
	"call": func(n int) string {
		return "local function f(x) return x end\nprint(" + strings.Repeat("f(", n) + "1" + strings.Repeat(")", n) + ")\n"
	},
	"function": func(n int) string {
		return "local f = " + strings.Repeat("function() return ", n/2) + "1" + strings.Repeat(" end", n/2) + "\nprint(type(f))\n"
	},
	"callback": func(n int) string {
		return "local function f(g) return g end\nf(" + strings.Repeat("function() f(", n/2) + "print" + strings.Repeat(") end", n/2) + ")\n"
	},
}

// Nesting that Lua loads compiles, and runs the same; nesting Lua refuses
// ("C stack overflow" past 200 levels) is refused with that reason, and
// quickly: checking and generating it used to take seconds, then minutes.
func TestNestingLimit(t *testing.T) {
	dir := t.TempDir()
	for name, gen := range nestings {
		ok := filepath.Join(dir, name+"_ok.lua")
		os.WriteFile(ok, []byte(gen(180)), 0o644)
		if msgs, code := compileJSON(t, ok, "lua54", filepath.Join(dir, name+"_ok.out.lua")); code != 0 {
			t.Errorf("%s at 180 levels: exit %d %v", name, code, msgs)
		} else {
			for _, lua := range []string{"lua5.4", "lua5.3"} {
				if path, err := exec.LookPath(lua); err == nil {
					if out, err := exec.Command(path, filepath.Join(dir, name+"_ok.out.lua")).CombinedOutput(); err != nil {
						t.Errorf("%s at 180 levels does not run under %s: %s", name, lua, out)
					}
				}
			}
		}

		deep := filepath.Join(dir, name+"_deep.lua")
		os.WriteFile(deep, []byte(gen(2000)), 0o644)
		start := time.Now()
		msgs, code := compileJSON(t, deep, "lua54", filepath.Join(dir, name+"_deep.out.lua"))
		if took := time.Since(start); took > 3*time.Second {
			t.Errorf("%s at 2000 levels took %v", name, took)
		}
		if code != 1 || len(msgs) == 0 || !strings.Contains(msgs[0], "nested more than 200 levels") {
			t.Errorf("%s at 2000 levels: exit %d %v", name, code, msgs)
		}
	}
}

// Shapes whose checking was exponential in depth compile quickly at any
// depth Lua loads: functions in tables (each walked once per path into
// it) and generic calls of generic calls (each argument checked twice).
func TestNestedShapesCompileQuickly(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"functions_in_tables.lua": "local t = " + strings.Repeat("{ function() return ", 60) + "1" + strings.Repeat(" end }", 60) + "\nprint(type(t))\n",
		"generic_calls.lunar":     "local function id<T>(x: T): T\n    return x\nend\nlocal v: number = " + strings.Repeat("id(", 150) + "1" + strings.Repeat(")", 150) + "\nprint(v)\n",
		// async( and await( are calls of names in Lua; trying each as an
		// async arrow and parsing it again doubled the time per level.
		"async_calls.lua": "local function async(f) return f end\nlocal v = " + strings.Repeat("async(function() return ", 40) + "1" + strings.Repeat(" end)", 40) + "\nprint(type(v))\n",
		// So did match(, tried as `match x with` first.
		"match_calls.lua": "local function match(x) return x end\nlocal v = " + strings.Repeat("match(", 40) + "1" + strings.Repeat(")", 40) + "\nprint(v)\n",
	}
	for name, src := range cases {
		file := filepath.Join(dir, name)
		os.WriteFile(file, []byte(src), 0o644)
		start := time.Now()
		msgs, code := compileJSON(t, file, "lua54", filepath.Join(dir, name+".out.lua"))
		if took := time.Since(start); code != 0 || took > 2*time.Second {
			t.Errorf("%s: exit %d in %v %v", name, code, took, msgs)
		}
	}
}
