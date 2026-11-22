package types

import "testing"

// A subclass of a class that implemented its parent's abstract methods
// inherits those implementations; it does not have to repeat them.
func TestAbstractMethodImplementedByIntermediateClass(t *testing.T) {
	for _, err := range checkSource(t, `
abstract class Actor
    abstract speak(): string
end
class Goblin extends Actor
    speak(): string
        return "grr"
    end
end
class Shaman extends Goblin
end
local s: Shaman = Shaman()
`) {
		t.Errorf("unexpected type error: %s", err.Message)
	}
}

func TestAbstractMethodStillRequired(t *testing.T) {
	if errs := checkSource(t, `
abstract class Actor
    abstract speak(): string
end
class Goblin extends Actor
end
`); len(errs) != 1 {
		t.Errorf("expected Goblin to be told to implement speak, got %v", messages(errs))
	}
}
