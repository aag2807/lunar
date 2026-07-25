package codegen

import "testing"

// items?[1] must not index a nil receiver.
func TestOptionalIndexGuardsNilReceiver(t *testing.T) {
	output := generate(t, `
function first(items: any): any
	return items?[1]
end
`)

	assertContains(t, output, "local __temp = items")
	assertContains(t, output, "__temp[1]")
	assertContains(t, output, "return nil")
}

// Plain indexing stays a plain index.
func TestPlainIndexIsUnchanged(t *testing.T) {
	output := generate(t, `
function first(items: any): any
	return items[1]
end
`)

	assertContains(t, output, "return items[1]")
	assertNotContains(t, output, "__temp")
}

// fn?.(args) must not call a nil callee.
func TestOptionalCallGuardsNilCallee(t *testing.T) {
	output := generate(t, `
function run(fn: any): any
	return fn?.(1, 2)
end
`)

	assertContains(t, output, "local __fn = fn")
	assertContains(t, output, "__fn(1, 2)")
	assertContains(t, output, "return nil")
}

func TestPlainCallIsUnchanged(t *testing.T) {
	output := generate(t, `
function run(fn: any): any
	return fn(1, 2)
end
`)

	assertContains(t, output, "return fn(1, 2)")
	assertNotContains(t, output, "__fn")
}
