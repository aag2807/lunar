package types

import (
	"os"
	"strings"
	"testing"
)

// The standard library declarations are part of the compiler: print is
// defined however and wherever lunar runs, with no stdlib folder nearby.
func TestEmbeddedStdlibNeedsNoFiles(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	for _, err := range NewChecker().Check(parseFile(t, "main.lunar", "print(\"hi\")\nlocal s: string = tostring(1)\n")) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// The declarations describe Lua 5.3 and 5.4, which have load and utf8 and
// no loadstring.
func TestStdlibDescribesLua53And54(t *testing.T) {
	for _, err := range NewChecker().Check(parseFile(t, "main.lunar", `
local chunk = load("return 1")
local n: number = utf8.len("abc")
local c: string = utf8.char(72, 105)
local v: any = select(2, "a", "b")
`)) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
	errs := NewChecker().Check(parseFile(t, "main.lunar", "local f = loadstring(\"return 1\")\n"))
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "loadstring") {
		t.Errorf("loadstring should be undefined, got %v", messages(errs))
	}
}

// warn exists only in Lua 5.4.
func TestWarnOnlyForLua54(t *testing.T) {
	src := "warn(\"low memory\")\n"
	for _, err := range NewCheckerWithOptions(Options{Target: "lua54"}).Check(parseFile(t, "main.lunar", src)) {
		t.Errorf("lua54: unexpected type error: %s", err.Message)
	}
	if errs := NewChecker().Check(parseFile(t, "main.lunar", src)); len(errs) != 1 {
		t.Errorf("lua53: expected warn to be undefined, got %v", messages(errs))
	}
}
