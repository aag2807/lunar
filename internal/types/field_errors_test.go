package types

import (
	"strings"
	"testing"
)

const enemyDecl = `
interface Stats
    atk: number
    def: number
end
interface Enemy
    name: string
    hp: number
    speed?: number
    stats: Stats
end
`

// The editor console has to say which field is wrong. Every case used to be
// "Cannot assign type '<table literal>' to variable 'e' of type 'Enemy'".
func singleError(t *testing.T, src string) *TypeError {
	t.Helper()
	errs := checkSource(t, enemyDecl+src)
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error, got %v", messages(errs))
	}
	return errs[0]
}

func expectMessage(t *testing.T, err *TypeError, want string) {
	t.Helper()
	if err.Message != want {
		t.Errorf("message = %q\nwant      %q", err.Message, want)
	}
}

func TestWrongFieldTypeNamesTheField(t *testing.T) {
	err := singleError(t, `local e: Enemy = { name = "imp", hp = "ten", stats = { atk = 1, def = 1 } }`)
	expectMessage(t, err, "Field 'hp' of 'Enemy' must be 'number', got 'string'")
	// enemyDecl ends on line 11; the literal is on line 12, "ten" at column 39.
	if err.Line != 12 || err.Column != 39 {
		t.Errorf("reported at %d:%d, want 12:39 (the value)", err.Line, err.Column)
	}
}

func TestMisspeltFieldSuggestsTheRealOne(t *testing.T) {
	err := singleError(t, `local e: Enemy = { name = "imp", hpp = 10, stats = { atk = 1, def = 1 } }`)
	expectMessage(t, err, "'hpp' is not a field of 'Enemy'. Did you mean 'hp'?")
	if err.Line != 12 || err.Column != 34 {
		t.Errorf("reported at %d:%d, want 12:34 (the key)", err.Line, err.Column)
	}
}

func TestUnknownFieldWithoutSuggestion(t *testing.T) {
	err := singleError(t, `local e: Enemy = { name = "imp", hp = 1, loot = 5, stats = { atk = 1, def = 1 } }`)
	expectMessage(t, err, "'loot' is not a field of 'Enemy'")
}

func TestMissingFieldsAreListed(t *testing.T) {
	err := singleError(t, `local e: Enemy = { name = "imp" }`)
	expectMessage(t, err, "Missing field(s) 'hp', 'stats' required by 'Enemy'")
	if err.Line != 12 || err.Column != 18 {
		t.Errorf("reported at %d:%d, want 12:18 (the '{')", err.Line, err.Column)
	}
}

func TestNestedFieldIsNamedByPath(t *testing.T) {
	err := singleError(t, `local e: Enemy = { name = "imp", hp = 1, stats = { atk = "3", def = 1 } }`)
	expectMessage(t, err, "Field 'stats.atk' of 'Enemy' must be 'number', got 'string'")
}

func TestNestedMissingField(t *testing.T) {
	err := singleError(t, `local e: Enemy = { name = "imp", hp = 1, stats = { atk = 3 } }`)
	expectMessage(t, err, "Missing field(s) 'stats.def' required by 'Enemy'")
}

func TestArrayElementFieldIsNamed(t *testing.T) {
	err := singleError(t, `local wave: Enemy[] = {
    { name = "a", hp = 1, stats = { atk = 1, def = 1 } },
    { name = "b", hp = "2", stats = { atk = 1, def = 1 } },
}`)
	expectMessage(t, err, "Field '[2].hp' of 'Enemy[]' must be 'number', got 'string'")
}

func TestFieldErrorsAtEveryCheckSite(t *testing.T) {
	cases := map[string]string{
		"argument": `function spawn(def: Enemy): void
end
spawn({ name = "imp", hp = "ten", stats = { atk = 1, def = 1 } })`,
		"return": `function make(): Enemy
    return { name = "imp", hp = "ten", stats = { atk = 1, def = 1 } }
end`,
		"assignment": `local e: Enemy = { name = "a", hp = 1, stats = { atk = 1, def = 1 } }
e = { name = "imp", hp = "ten", stats = { atk = 1, def = 1 } }`,
		"constructor": `class Spawner
    constructor(def: Enemy)
    end
end
local s: Spawner = Spawner({ name = "imp", hp = "ten", stats = { atk = 1, def = 1 } })`,
	}
	for site, src := range cases {
		errs := checkSource(t, enemyDecl+src)
		if len(errs) != 1 || errs[0].Message != "Field 'hp' of 'Enemy' must be 'number', got 'string'" {
			t.Errorf("%s: got %v", site, messages(errs))
		}
	}
}

func TestValidLiteralHasNoErrors(t *testing.T) {
	for _, err := range checkSource(t, enemyDecl+`local e: Enemy = { name = "imp", hp = 1, speed = 2, stats = { atk = 1, def = 1 } }`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

// Ordinary assignability problems keep their message.
func TestNonLiteralMismatchKeepsGenericMessage(t *testing.T) {
	errs := checkSource(t, enemyDecl+`local n: number = "x"`)
	if len(errs) != 1 || !strings.HasPrefix(errs[0].Message, "Cannot assign type") {
		t.Fatalf("got %v", messages(errs))
	}
}
