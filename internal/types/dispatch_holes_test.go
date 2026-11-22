package types

import (
	"strings"
	"testing"
)

// When the types in a union disagree on whether a member is a method or a
// plain field, no single call works for all of them.
func TestUnionDisagreeingOnDispatchIsAnError(t *testing.T) {
	errs := checkSource(t, `
class A
    hit(n: number): number
        return n
    end
end
interface B
    hit: function(n: number): number
end
local function use(x: A | B): number
    return x.hit(3)
end
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'hit' is a method of 'A' but a plain field of 'B'") {
		t.Fatalf("got %v", messages(errs))
	}
}

// A union member that lacks the member is reported.
func TestUnionMemberMissingIsAnError(t *testing.T) {
	errs := checkSource(t, `
class A
    hit(n: number): number
        return n
    end
end
class C
end
local function use(x: A | C): number
    return x.hit(3)
end
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'C' has no member 'hit'") {
		t.Fatalf("got %v", messages(errs))
	}
}

// Read without a call, a method loses its receiver: f(3) would pass 3 as
// self. It has to be called where it is named.
func TestMethodReadAsValueIsAnError(t *testing.T) {
	errs := checkSource(t, `
class Enemy
    hp: number = 10
    hit(n: number): number
        self.hp = self.hp - n
        return self.hp
    end
end
local e = Enemy()
local f = e.hit
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "Method 'hit' of 'Enemy' has to be called") {
		t.Fatalf("got %v", messages(errs))
	}
}

// Calls, explicit ':' calls and plain fields are unaffected.
func TestMethodCallsStillCheck(t *testing.T) {
	for _, err := range checkSource(t, `
class Enemy
    hp: number = 10
    onDeath: function(): void = function() end
    hit(n: number): number
        return self.hp - n
    end
end
local e = Enemy()
local a: number = e.hit(1)
local b: number = e:hit(2)
local g = e.onDeath
g()
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// A class value fits a union that lists its class (or a parent of it).
func TestClassIsAssignableToAUnionContainingIt(t *testing.T) {
	for _, err := range checkSource(t, `
class A
end
class B
end
class C extends A
end
local x: A | B = A()
local y: A | B = C()
local z: B? = B()
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
