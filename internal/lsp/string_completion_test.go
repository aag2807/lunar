package lsp

import "testing"

// Inside the quotes of an argument whose parameter is a union of string
// literals, completion offers those strings: an inline union, an alias, a
// later argument, a method's, and a string still being typed. Other
// strings get nothing.
func TestCompletionInsideQuotesOffersTheUnionsStrings(t *testing.T) {
	head := "type Mood = \"calm\" | \"angry\"\n" +
		"local function setKind(kind: \"sentry\" | \"brute\" | \"warden\", mood?: Mood): void\nend\n" +
		"class Guard\n    face(mood: Mood): void\n    end\nend\nlocal g = Guard()\n"
	cases := []struct {
		text, marker string
		want         []string
	}{
		{head + "setKind(\"br\")\n", "setKind(\"br", []string{"sentry", "brute", "warden"}},
		{head + "setKind(\"sentry\", \"\")\n", "setKind(\"sentry\", \"", []string{"calm", "angry"}},
		{head + "g:face(\"\")\n", "g:face(\"", []string{"calm", "angry"}},
		{head + "setKind(\"", "setKind(\"", []string{"sentry", "brute", "warden"}},
	}
	for _, tc := range cases {
		labels := completeAt(t, tc.text, tc.marker)
		if !has(labels, tc.want...) || len(labels) != len(tc.want) {
			t.Errorf("at %q: got %v, want %v", tc.marker, labels, tc.want)
		}
	}
	if labels := completeAt(t, head+"print(\"\")\n", "print(\""); len(labels) != 0 {
		t.Errorf("in print's string: got %v", labels)
	}
}

// The quotes trigger completion, so that the strings show as one is
// opened; right after a closing quote, where no name can follow, nothing
// is offered.
func TestQuotesTriggerCompletion(t *testing.T) {
	if labels := completeAt(t, "local x = 1\nprint(\"hi\")\n", "print(\"hi\""); len(labels) != 0 {
		t.Errorf("after a closing quote: got %v", labels)
	}
	c := newClient(t)
	result := decode[map[string]any](t, c.request("initialize", map[string]any{"capabilities": map[string]any{}}))
	triggers := result["capabilities"].(map[string]any)["completionProvider"].(map[string]any)["triggerCharacters"]
	got := map[string]bool{}
	for _, ch := range triggers.([]any) {
		got[ch.(string)] = true
	}
	if !got["\""] || !got["'"] {
		t.Errorf("trigger characters %v", triggers)
	}
}
