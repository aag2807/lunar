package examples

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The engine side's checks, kept here so they run on every commit: a
// project shaped like the engine's, plain-Lua snippets with Lua's answers,
// and wrong code that must be refused naming the problem.

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, text := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// buildJSON runs lunar build --json in dir and returns the exit code, the
// diagnostics' messages and the summary.
func buildJSON(t *testing.T, dir string) (int, []string, map[string]any) {
	t.Helper()
	cmd := exec.Command(compilerBin, "build", "--json")
	cmd.Dir = dir
	out, err := cmd.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
		if len(exit.Stderr) > 0 {
			t.Errorf("stderr in JSON mode: %s", exit.Stderr)
		}
	}
	var msgs []string
	var summary map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var v map[string]any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			t.Fatalf("not JSON: %q", line)
		}
		if v["kind"] == "summary" {
			summary = v
		} else {
			msgs = append(msgs, v["message"].(string))
		}
	}
	return code, msgs, summary
}

func lua54(t *testing.T) string {
	t.Helper()
	for _, interp := range interpreters(t) {
		if interp.version == "5.4" {
			return interp.path
		}
	}
	if os.Getenv("LUNAR_REQUIRE_LUA") == "1" {
		t.Fatal("needs Lua 5.4")
	}
	t.Skip("no Lua 5.4")
	return ""
}

const engineBoss = `interface Phase
    kind: "chase" | "armour" | "duel"
    label: string
    health: number
    speed: number?
end
interface Boss
    name: string
    wake: number
    phases: Phase[]
end
const warden: Boss = {
    name = "warden",
    wake = 30,
    phases = {
        { kind = "chase", label = "CATCH IT", health = 3, speed = 22 },
        { kind = "duel", label = "FIGHT", health = 6 },
    },
}
return warden
`

func engineProject(t *testing.T) string {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"engine/api/surge.d.lunar": `declare interface SpringDef
    at: number[]
    dir: number[]
    speed: number
end
declare function ring(at: number[]): number
declare function spring(def: SpringDef): number
declare function message(text: string): void
declare function on_ground(x: number, z: number, lift: number): number[]
`,
		"game/lunar.config.json": `{ "rootDir": "scripts", "outDir": "../build/scripts", "target": "lua54",
  "declarations": ["../engine/api"], "dataFiles": ["data/**"], "sourceMap": true }`,
		"game/scripts/data/boss.lunar": engineBoss,
		"game/scripts/level.lunar": `class Counter
    n: number
    constructor(start: number)
        self.n = start
    end
    bump(by: number): number
        self.n = self.n + by
        return self.n
    end
end
local c = Counter.new(1)
local d = Counter(10)
for i = 0, 3 do
    ring({ i * 2, 1, -5 })
end
local s = spring({ at = on_ground(4, -8, 0.2), dir = { 0, 1, 0 }, speed = 20 })
message("spring " .. s .. " count " .. (c:bump(2) + d:bump(1)))
`,
	})
	return root
}

// Section 2: lunar build on a project shaped like the engine's.
func TestEngineShapedProject(t *testing.T) {
	lua := lua54(t)
	root := engineProject(t)
	game, out := filepath.Join(root, "game"), filepath.Join(root, "build", "scripts")

	code, msgs, summary := buildJSON(t, game)
	if code != 0 || len(msgs) != 0 || summary["errors"] != 0.0 || summary["files"] != 2.0 || summary["written"] != true {
		t.Fatalf("build: exit %d, %v, %v", code, msgs, summary)
	}
	var files []string
	filepath.Walk(out, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(out, path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	if strings.Join(files, " ") != ".lunar-manifest.json data/boss.lua data/boss.lua.map level.lua level.lua.map" {
		t.Errorf("build folder holds %v", files)
	}

	boss := readFile(t, filepath.Join(out, "data", "boss.lua"))
	if n, w, p := strings.Index(boss, "name ="), strings.Index(boss, "wake ="), strings.Index(boss, "phases ="); !(n >= 0 && n < w && w < p) {
		t.Errorf("boss.lua fields are not in source order:\n%s", boss)
	}

	// The data file in a Lua environment with no globals at all.
	check := `local f = assert(loadfile(arg[1], "t", {}))
local b = f()
print(b.phases[2].label, b.phases[2].speed, b.phases[1].speed)`
	if got := runLuaCode(t, lua, check, filepath.Join(out, "data", "boss.lua")); got != "FIGHT\tnil\t22\n" {
		t.Errorf("boss.lua returned %q", got)
	}

	// level.lua against a stand-in engine.
	stub := `function ring(at) return 1 end
function spring(def) print("spring speed", def.speed) return 7 end
function on_ground(x, z, lift) return { 1, 2, 3 } end
function message(text) print("message", text) end
dofile(arg[1])`
	if got := runLuaCode(t, lua, stub, filepath.Join(out, "level.lua")); got != "spring speed\t20\nmessage\tspring 7 count 14\n" {
		t.Errorf("level.lua printed %q", got)
	}
}

// Section 4.4: a value outside a literal union, in a data file; the last
// good build stays.
func TestEngineLiteralUnionIsRefused(t *testing.T) {
	root := engineProject(t)
	game := filepath.Join(root, "game")
	if code, msgs, _ := buildJSON(t, game); code != 0 {
		t.Fatalf("first build: %v", msgs)
	}
	before := readFile(t, filepath.Join(root, "build", "scripts", "data", "boss.lua"))
	writeTree(t, game, map[string]string{"scripts/data/boss.lunar": strings.Replace(engineBoss, `kind = "duel"`, `kind = "dool"`, 1)})
	code, msgs, summary := buildJSON(t, game)
	if code != 1 || summary["written"] != false || len(msgs) == 0 || !strings.Contains(strings.Join(msgs, "\n"), "kind") {
		t.Errorf("exit %d, %v", code, msgs)
	}
	if after := readFile(t, filepath.Join(root, "build", "scripts", "data", "boss.lua")); after != before {
		t.Error("the refused build changed the last good boss.lua")
	}
}

func runLuaCode(t *testing.T, lua, code, file string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "run.lua")
	os.WriteFile(script, []byte(code), 0o644)
	out, err := exec.Command(lua, script, file).CombinedOutput()
	if err != nil {
		t.Fatalf("lua: %v\n%s", err, out)
	}
	return string(out)
}

// Section 3: snippets that compile with no diagnostics and print Lua's
// answers.
func TestEngineSnippetsGiveLuasAnswers(t *testing.T) {
	lua := lua54(t)
	for name, c := range map[string]struct{ src, want string }{
		"3.1 precedence": {`local n: number = 5
local s: string = "left " .. n - 1
print(s, (s:len() + 1) * 2, -2 ^ 2, 2 ^ 3 ^ 2, 2 * 3 ^ 2, type "x")
print((1 + 2) * 3)
print(2 ^ (1 + 1), -(2 + 3), not (1 == 2))
local a = false
print("x" .. 2 * 3, 1 + 2 .. "a", "a" .. "b" == "ab", not a == false, 1 << 4 | 1, #"abc" + 1, 7 // 2, 7 % 3)
`, "left 4\t14\t-4.0\t512.0\t18.0\tstring\n9\n4.0\t-5\ttrue\nx6\t3a\ttrue\tfalse\t17\t4\t3\t1\n"},
		"3.2 words": {`local t = { from = 1, set = 2, class = 3; 4 }
for _, p in ipairs { 10, 20 } do print(p, t.from + t.set + t.class) end
`, "10\t6\n20\t6\n"},
		"3.3 call forms": {`declare function enemy(def: any): void
local function enemy_(def) end
for _, place in ipairs { { -12, 197 }, { -15, 200 } } do print(place[1]) end
print(type "x")
local s: string = "abc"
print((s:upper()), ("x"):rep(3))
`, "-12\n-15\nstring\nABC\txxx\n"},
		"3.4 and/or": {`local c = {}
c.attempt = (c.attempt or 0) + 1
local a = nil
local b = a or 5
print(c.attempt, b + 1, (1 > 2) and "yes" or "no")
`, "1\t6\tno\n"},
		"3.5 other Lua": {`local t = { [1] = "a"; "b" }
goto done
::done::
local x <const> = 5
print(t[1], x // 2, 3 ~= 4, ~5)
`, "b\t2\ttrue\t-6\n"},
	} {
		dir := t.TempDir()
		writeTree(t, dir, map[string]string{"case.lunar": c.src})
		cmd := exec.Command(compilerBin, "--json", "--target", "lua54", "case.lunar")
		cmd.Dir = dir
		out, err := cmd.Output()
		if err != nil || strings.Count(strings.TrimSpace(string(out)), "\n") != 0 {
			t.Errorf("%s: compile: %v\n%s", name, err, out)
			continue
		}
		got, err := exec.Command(lua, filepath.Join(dir, "case.lua")).CombinedOutput()
		if err != nil || string(got) != c.want {
			t.Errorf("%s: printed %q, want %q (%v)", name, got, c.want, err)
		}
	}
}

// Section 3.2: every word the engine side lists works as a local, a
// parameter and a field; super and self as fields.
func TestEngineWordsAsNames(t *testing.T) {
	lua := lua54(t)
	words := strings.Fields(`set get from match with is as class enum interface namespace const readonly declare import
export async await abstract static extends implements keyof typeof constructor public private
protected number type new kind name value key data target object string table`)
	var src strings.Builder
	for i, w := range words {
		src.WriteString("do\n")
		src.WriteString("    local " + w + " = " + strconv.Itoa(i) + "\n")
		src.WriteString("    local function f" + strconv.Itoa(i) + "(" + w + ") return " + w + " end\n")
		src.WriteString("    local t = { " + w + " = " + w + " }\n")
		src.WriteString("    t." + w + " = f" + strconv.Itoa(i) + "(t." + w + ")\n")
		src.WriteString("    io.write(t." + w + ", \" \")\n")
		src.WriteString("end\n")
	}
	src.WriteString("local u = { super = 1, self = 2 }\nprint(u.super + u.self)\n")
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"words.lunar": src.String()})
	cmd := exec.Command(compilerBin, "--json", "--target", "lua54", "words.lunar")
	cmd.Dir = dir
	if out, err := cmd.Output(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	got, err := exec.Command(lua, filepath.Join(dir, "words.lua")).CombinedOutput()
	var want strings.Builder
	for i := range words {
		want.WriteString(strconv.Itoa(i) + " ")
	}
	want.WriteString("3\n")
	if err != nil || string(got) != want.String() {
		t.Errorf("printed %q, want %q (%v)", got, want.String(), err)
	}
}

// Section 4: wrong code is refused, naming the problem, and nothing is
// written.
func TestEngineWrongCodeIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		src  string
		want []string
	}{
		"4.1 literals against an interface": {`interface P
    name: string
    health: number
    speed: number?
end
const a: P = { name = "x", helth = 3 }
const b: P = { name = "x", health = "three" }
const c: P = { name = "x" }
const d: P = { name = "x", health = 1 }
`, []string{"'helth' is not a field of 'P'. Did you mean 'health'?",
			"Field 'health' of 'P' must be 'number', got 'string'",
			"Missing field(s) 'health' required by 'P'"}},
		"4.2 calls to declared functions": {`declare function ring(at: number[]): number
declare function message(text: string): void
message(ring("no"))
`, []string{"to parameter of type 'number[]'", "to parameter of type 'string'"}},
		"4.3 annotated code stays strict": {`interface P
    name: string
    health: number
end
local function hurt(p: P, by: number): number
    return p.health - by
end
hurt({ name = "x", helth = 3 }, 1)
hurt({ name = "x", health = 3 })
local n: number = "five"
`, []string{"'helth' is not a field of 'P'", "expects at least 2 arguments", "variable 'n' of type 'number'"}},
	} {
		dir := t.TempDir()
		writeTree(t, dir, map[string]string{"wrong.lunar": c.src})
		cmd := exec.Command(compilerBin, "--json", "wrong.lunar")
		cmd.Dir = dir
		out, err := cmd.Output()
		exit, _ := err.(*exec.ExitError)
		if exit == nil || exit.ExitCode() != 1 {
			t.Errorf("%s: want exit 1, got %v", name, err)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "wrong.lua")); statErr == nil {
			t.Errorf("%s: wrong.lua was written", name)
		}
		var msgs []string
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			var v struct{ Kind, Message string }
			json.Unmarshal([]byte(line), &v)
			if v.Kind == "diagnostic" {
				msgs = append(msgs, v.Message)
			}
		}
		if len(msgs) != len(c.want) {
			t.Errorf("%s: want %d errors, got %d: %v", name, len(c.want), len(msgs), msgs)
		}
		joined := strings.Join(msgs, "\n")
		for _, w := range c.want {
			if !strings.Contains(joined, w) {
				t.Errorf("%s: missing %q in:\n%s", name, w, joined)
			}
		}
	}
}
