package types

import (
	"strings"
	"testing"
)

// The owner's first engine declarations: the first function has no 'end'.
// That used to swallow despawn and tick, which were then reported as
// undefined at their call sites.
func TestDeclarationsWithoutEndAreAllDefined(t *testing.T) {
	input := `
declare function spawn(name: string): number
declare function despawn(id: number): void end
declare function tick(dt: number): void
local id: number = spawn("goblin")
despawn(id)
tick(0.016)
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestDeclaredClassIsUsable(t *testing.T) {
	input := `
declare class Sprite
    x: number
    constructor(path: string)
    draw(): void
    static load(path: string): Sprite
end
local s: Sprite = Sprite.load("hero.png")
s.draw()
local n: number = s.x
local t: Sprite = Sprite("tree.png")
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestDeclaredNamespaceMembers(t *testing.T) {
	input := `
declare namespace audio
    function play(name: string, volume?: number): void
    const masterVolume: number
end
audio.play("boom")
audio.play("boom", 0.5)
local v: number = audio.masterVolume
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestDeclaredNamespaceRejectsUnknownMember(t *testing.T) {
	errs := checkSource(t, `
declare namespace audio
    function play(name: string): void
end
audio.plya("boom")
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'plya'") || !strings.Contains(errs[0].Message, "'play'") {
		t.Fatalf("expected one error naming plya and suggesting play, got %v", messages(errs))
	}
}

// Members of a declared namespace are reached through it, not as globals.
func TestDeclaredNamespaceMembersAreNotGlobals(t *testing.T) {
	errs := checkSource(t, `
declare namespace audio
    function play(name: string): void
end
play("boom")
`)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "'play'") {
		t.Fatalf("expected an undefined-variable error for play, got %v", messages(errs))
	}
}

// A declared interface may name itself, or one declared after it; C APIs
// return handles to their own type all the time.
func TestDeclaredInterfacesMayReferToThemselvesAndEachOther(t *testing.T) {
	input := `
declare interface DigestContext
    update: function(data: string): DigestContext
    hex: function(): string
end
declare interface Node
    next: Edge?
end
declare interface Edge
    target: Node
end
declare const digest: DigestContext
local d: DigestContext = digest.update("a").update("b")
local s: string = d.hex()
`
	for _, err := range checkSource(t, input) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}
