package project

import (
	"testing"

	"lunar/internal/types"
)

// A checker error with no file (raised on a token the parser did not
// produce) belongs to the module being checked; errors in other files
// (declarations, reported once on their own) are left out.
func TestModuleDiagnosticsKeepErrorsWithoutAFile(t *testing.T) {
	errs := []*types.TypeError{
		{Message: "own", File: "/s/main.lunar", Line: 3, Column: 2},
		{Message: "no file"},
		{Message: "declaration", File: "/s/api.d.lunar", Line: 1, Column: 1},
	}
	got := moduleDiags(errs, "/s/main.lunar")
	if len(got) != 2 || got[0].Message != "own" || got[1].Message != "no file" {
		t.Fatalf("got %+v", got)
	}
	if got[1].File != "/s/main.lunar" || got[1].Line != 1 || got[1].Column != 1 {
		t.Errorf("file-less error placed at %s:%d:%d", got[1].File, got[1].Line, got[1].Column)
	}
}
