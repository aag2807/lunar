package types

import (
	"strings"
	"testing"
)

func TestGenericInterfaceInstances(t *testing.T) {
	for _, err := range checkSource(t, `
interface Box<T>
    value: T
    get(): T
end
interface Node<T>
    value: T
    next: Node<T>?
end
local b: Box<number> = { value = 1, get = function(self): number
    return self.value
end }
local n: number = b.value + b.get()
local s: Box<string> = { value = "a", get = function(self): string
    return "a"
end }
local name: string = s.value
local list: Node<number> = { value = 1, next = { value = 2 } }
local second: number = list.next.value
local nested: Box<Box<number>>
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestGenericInterfaceErrors(t *testing.T) {
	errs := checkSource(t, `
interface Box<T>
    value: T
end
local a: Box<number> = { value = "one" }
local b: string = (a as Box<number>).value
local c: Box<number, string>
local d: Box
`)
	got := strings.Join(messages(errs), "\n")
	for _, want := range []string{
		"Field 'value' of 'Box<number>' must be 'number', got 'string'",
		"Cannot assign type 'number' to variable 'b' of type 'string'",
		"Generic type 'Box' expects 1 type argument(s), got 2",
		"Generic type 'Box' needs type arguments: Box<...>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}
