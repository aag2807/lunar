package project

import (
	"strings"
	"testing"
)

// An exported async function is async in the importing file too: calling
// it without await is an error there, with the code tools rely on.
func TestImportedAsyncFunctionMustBeAwaited(t *testing.T) {
	root := tree(t, map[string]string{
		"lunar.config.json": `{"outDir": "../build"}`,
		"time.lunar":        "export async function wait(n: number): number\n    return coroutine.yield(n)\nend\n",
		"main.lunar": "import { wait } from \"./time\"\n" +
			"export async function patrol(): void\n    local n: number = await wait(1)\nend\n" +
			"function update(): void\n    wait(2)\nend\n",
	})
	_, diags := open(t, Options{Dir: root}).Check()
	if len(diags) != 1 {
		t.Fatalf("want one diagnostic, got %v", messagesOf(diags))
	}
	d := diags[0]
	if d.Code != "async" || !strings.Contains(d.Message, "'wait' can pause") || !strings.HasSuffix(d.File, "main.lunar") || d.Line != 6 || d.Column != 5 {
		t.Errorf("got %s %q (%s)", d.Location(), d.Message, d.Code)
	}
}
