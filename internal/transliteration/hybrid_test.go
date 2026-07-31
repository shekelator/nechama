package transliteration

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// stubProvider is a minimal Provider that records the prompt and returns a
// canned response (or error). It lets the hybrid tests observe exactly what
// the service asks the LLM and inject a controlled answer.
type stubProvider struct {
	resp       string
	err        error
	gotSystem  string
	gotUser    string
	calls      int
}

func (s *stubProvider) Generate(_ context.Context, system, user string) (string, error) {
	s.calls++
	s.gotSystem = system
	s.gotUser = user
	return s.resp, s.err
}

// TestHybridNilProviderUsesEngineOnly verifies that without a provider the
// hybrid is a pure deterministic engine: ambiguous words keep the engine's
// best-guess transliteration and no LLM is consulted.
func TestHybridNilProviderUsesEngineOnly(t *testing.T) {
	t.Parallel()

	svc, err := NewHybridService(nil, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "שְׁמַע מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Shema midbar" {
		t.Fatalf("got %q want %q", out, "Shema midbar")
	}
}

// TestHybridNoAmbiguousNoLLMCall verifies that when the engine flags nothing
// ambiguous, the provider is never called.
func TestHybridNoAmbiguousNoLLMCall(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: "{}"}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "שְׁמַע"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Shema" {
		t.Fatalf("got %q want %q", out, "Shema")
	}
	if stub.calls != 0 {
		t.Fatalf("provider called %d times, want 0", stub.calls)
	}
}

// TestHybridConsultsLLMOnlyForAmbiguous verifies the provider is asked for
// exactly the flagged words (מִדְבָּר) and not for the unambiguous ones
// (שְׁמַע), and that the answer is spliced back.
func TestHybridConsultsLLMOnlyForAmbiguous(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: `{"מִדְבָּר":"midbar"}`}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "שְׁמַע מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Shema midbar" {
		t.Fatalf("got %q want %q", out, "Shema midbar")
	}
	if stub.calls != 1 {
		t.Fatalf("provider called %d times, want 1", stub.calls)
	}
	if !strings.Contains(stub.gotUser, "מִדְבָּר") {
		t.Fatalf("user prompt should include the ambiguous word, got %q", stub.gotUser)
	}
	if strings.Contains(stub.gotUser, "שְׁמַע") {
		t.Fatalf("user prompt should not include the unambiguous word, got %q", stub.gotUser)
	}
	if !strings.Contains(stub.gotSystem, "HEBREW TRANSLITERATION RULES") {
		t.Fatalf("system prompt should embed the rules, got %q", stub.gotSystem)
	}
}

// TestHybridSplicesDistinctReplacement verifies the LLM answer actually
// replaces the engine's best guess when it differs.
func TestHybridSplicesDistinctReplacement(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: `{"מִדְבָּר":"midhbar"}`}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "שְׁמַע מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Shema midhbar" {
		t.Fatalf("got %q want %q", out, "Shema midhbar")
	}
}

// TestHybridRepeatedWordAskedOnce verifies a repeated ambiguous word is sent
// to the LLM once and both occurrences are replaced.
func TestHybridRepeatedWordAskedOnce(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: `{"מִדְבָּר":"midbar"}`}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "מִדְבָּר מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Midbar midbar" {
		t.Fatalf("got %q want %q", out, "Midbar midbar")
	}
	if stub.calls != 1 {
		t.Fatalf("provider called %d times, want 1", stub.calls)
	}
}

// TestHybridCapitalizesLineStartReplacement verifies LLM replacements that
// land at the start of a line are capitalized.
func TestHybridCapitalizesLineStartReplacement(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: `{"מִדְבָּר":"midbar"}`}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "מִדְבָּר\nמִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Midbar\nMidbar" {
		t.Fatalf("got %q want %q", out, "Midbar\nMidbar")
	}
}

// TestHybridProviderErrorFallsBack verifies a provider error is swallowed and
// the engine's best-guess output is returned instead.
func TestHybridProviderErrorFallsBack(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{err: errors.New("network down")}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Midbar" {
		t.Fatalf("got %q want %q", out, "Midbar")
	}
}

// TestHybridGarbageResponseFallsBack verifies a non-JSON response falls back
// to the engine best-guess rather than erroring.
func TestHybridGarbageResponseFallsBack(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: "I am not a JSON object"}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Midbar" {
		t.Fatalf("got %q want %q", out, "Midbar")
	}
}

// TestHybridCodeFenceResponse verifies ```json code fences around the map are
// stripped before parsing.
func TestHybridCodeFenceResponse(t *testing.T) {
	t.Parallel()

	stub := &stubProvider{resp: "```json\n{\"מִדְבָּר\":\"midbar\"}\n```"}
	svc, err := NewHybridService(stub, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}

	out, err := svc.Transliterate(context.Background(), Request{Text: "מִדְבָּר"})
	if err != nil {
		t.Fatalf("Transliterate() error = %v", err)
	}
	if out != "Midbar" {
		t.Fatalf("got %q want %q", out, "Midbar")
	}
}

// TestHybridEmptyInput verifies empty input is rejected.
func TestHybridEmptyInput(t *testing.T) {
	t.Parallel()

	svc, err := NewHybridService(nil, DefaultRules, nil)
	if err != nil {
		t.Fatalf("NewHybridService() error = %v", err)
	}
	if _, err := svc.Transliterate(context.Background(), Request{Text: ""}); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

// TestParseTransliterationMap exercises the JSON-extraction helper directly.
func TestParseTransliterationMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		resp    string
		want    map[string]string
		wantErr bool
	}{
		{"plain object", `{"a":"b","c":"d"}`, map[string]string{"a": "b", "c": "d"}, false},
		{"json fence", "```json\n{\"a\":\"b\"}\n```", map[string]string{"a": "b"}, false},
		{"plain fence", "```\n{\"a\":\"b\"}\n```", map[string]string{"a": "b"}, false},
		{"prose around object", `sure, here it is: {"a":"b"} hope that helps`, map[string]string{"a": "b"}, false},
		{"no object", "not json at all", nil, true},
		{"empty", "", nil, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseTransliterationMap(tc.resp)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %d want %d (%v)", len(got), len(tc.want), got)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Fatalf("key %q: got %q want %q", k, got[k], v)
				}
			}
		})
	}
}