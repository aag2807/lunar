package types

import (
	"strings"
	"testing"
)

// async/await compiled to a coroutine that nothing ever resumed, so the body
// never ran. Until there is a scheduler to run them, both are refused.
func TestAsyncFunctionIsRefused(t *testing.T) {
	errs := checkSource(t, `
async function fetchData(): string
	return "data"
end
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'async' functions are not supported") ||
		!strings.Contains(errs[0].Message, "coroutine") {
		t.Fatalf("got %v", messages(errs))
	}
}

func TestAwaitIsRefused(t *testing.T) {
	errs := checkSource(t, `
function load(): any
	return nil
end
function run(): void
	local x = await load()
end
`)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "'await' is not supported") {
			found = true
		}
	}
	if !found {
		t.Fatalf("got %v", messages(errs))
	}
}
