// Package host builds a tiny C program that loads a Lua chunk the way the
// engine loads data files: luaL_newstate() with no libraries opened. A data
// file that compiles but touches any global fails here, not in the engine.
package host

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// luaPackages are the pkg-config names tried for each Lua version.
var luaPackages = []string{"lua5.4", "lua5.3"}

// buildHosts compiles csrc/nolibs.c once per Lua version whose headers and
// library pkg-config can find.
func buildHosts(t *testing.T) map[string]string {
	t.Helper()
	cc, ccErr := exec.LookPath("cc")
	hosts := map[string]string{}
	for _, pkg := range luaPackages {
		if ccErr != nil {
			break
		}
		flags, err := exec.Command("pkg-config", "--cflags", "--libs", pkg).Output()
		if err != nil {
			continue
		}
		bin := filepath.Join(t.TempDir(), "nolibs-"+pkg)
		args := append([]string{"-std=c99", "-Wall", "-Wextra", "-Werror", "-o", bin, filepath.Join("csrc", "nolibs.c")}, strings.Fields(string(flags))...)
		if out, err := exec.Command(cc, args...).CombinedOutput(); err != nil {
			t.Fatalf("building nolibs.c against %s: %v\n%s", pkg, err, out)
		}
		hosts[pkg] = bin
	}
	if len(hosts) == 0 {
		if os.Getenv("LUNAR_REQUIRE_LUA") == "1" {
			t.Fatal("LUNAR_REQUIRE_LUA=1 but no C compiler with Lua development files was found")
		}
		t.Skip("no C compiler with Lua development files (install liblua5.4-dev)")
	}
	return hosts
}

func runHost(bin, file string) (stdout, stderr string, err error) {
	var o, e bytes.Buffer
	cmd := exec.Command(bin, file)
	cmd.Stdout, cmd.Stderr = &o, &e
	err = cmd.Run()
	return o.String(), e.String(), err
}

func TestHostDumpsReturnedTableDeterministically(t *testing.T) {
	want, err := os.ReadFile("testdata/enemy_data.expected")
	if err != nil {
		t.Fatal(err)
	}
	for pkg, bin := range buildHosts(t) {
		got, stderr, err := runHost(bin, "testdata/enemy_data.lua")
		if err != nil {
			t.Fatalf("%s: %v\n%s", pkg, err, stderr)
		}
		if got != string(want) {
			t.Errorf("%s: dump differs\n--- want\n%s--- got\n%s", pkg, want, got)
		}
	}
}

// The host must not open the standard library, or it would hide exactly the
// mistake it exists to catch.
func TestHostRejectsChunkThatUsesALibrary(t *testing.T) {
	for pkg, bin := range buildHosts(t) {
		_, stderr, err := runHost(bin, "testdata/uses_library.lua")
		if err == nil {
			t.Errorf("%s: a chunk calling tostring loaded without error", pkg)
		}
		if !strings.Contains(stderr, "tostring") {
			t.Errorf("%s: stderr should name the missing global, got %q", pkg, stderr)
		}
	}
}

func TestHostRejectsChunkThatDoesNotReturnATable(t *testing.T) {
	for pkg, bin := range buildHosts(t) {
		_, stderr, err := runHost(bin, "testdata/not_a_table.lua")
		if err == nil {
			t.Errorf("%s: a chunk returning 42 was accepted", pkg)
		}
		if !strings.Contains(stderr, "must return a table") {
			t.Errorf("%s: unexpected stderr %q", pkg, stderr)
		}
	}
}
