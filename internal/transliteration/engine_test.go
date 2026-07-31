package transliteration

import (
	"strings"
	"testing"
)

// TestEngineGoldenExamples runs the engine against every example in the
// Examples table of rules.go. The engine (deterministic, no model) must
// reproduce each expected output exactly. These are the canonical cases the
// whole transliteration pipeline is expected to satisfy.
func TestEngineGoldenExamples(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)

	cases := []struct {
		name   string
		hebrew string
		want   string
	}{
		{"leading shva + shin", "שְׁמַע", "shema"},
		{"silent shva closes syllable", "מִדְבָּר", "midbar"},
		{"dagesh chazak single consonant", "שַׁבָּת", "shabat"},
		{"definite article", "הַבַּיִת", "ha-bayit"},
		{"silent final he + sin dot", "שָׂדֶה", "sade"},
		{"cholam male", "טוֹב", "tov"},
		{"bare cholam", "אֹמֶר", "omer"},
		{"bare cholam + shin", "קֹדֶשׁ", "kodesh"},
		{"tsere + final tsadi", "עֵץ", "ets"},
		{"inseparable preposition bet", "בְּיוֹם", "be-yom"},
		{"vav shva prefix + silent shva", "וְאָהַבְתָּ", "ve-ahavta"},
		{"word-initial bet with dagesh", "בָּשָׂר", "basar"},
		{"word-initial bet without dagesh", "בָא", "va"},
		{"bet without dagesh after vowel", "אָבִיב", "aviv"},
		{"word-initial kaf with dagesh", "כָּבוֹד", "kavod"},
		{"word-initial kaf without dagesh", "כִי", "chi"},
		{"final kaf without dagesh", "אַךְ", "ach"},
		{"divine name", "יְהוָה", "ADONAI"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _, _ := eng.transliterateWord(tc.hebrew, false)
			if got != tc.want {
				t.Fatalf("hebrew=%q\n got %q\nwant %q", tc.hebrew, got, tc.want)
			}
		})
	}
}

// TestEngineTransliterateSegments checks the full segment stream: spacing,
// newlines, capitalization, and the Hebrew/Ambiguous metadata on segments.
func TestEngineTransliterateSegments(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)

	// "Shalom alechem" — first word capitalized, second not.
	segs := eng.Transliterate("שָׁלוֹם עָלֵיכֶם")
	if got := joinText(segs); got != "Shalom alechem" {
		t.Fatalf("joined: got %q want %q", got, "Shalom alechem")
	}
	// First segment is the capitalized word; second is " "; third is the word.
	if !segs[0].LineStart {
		t.Fatalf("first segment should be LineStart, got %+v", segs[0])
	}
	if segs[0].Hebrew != "שָׁלוֹם" {
		t.Fatalf("first segment Hebrew: got %q want %q", segs[0].Hebrew, "שָׁלוֹם")
	}
	if segs[1].Text != " " || segs[1].Hebrew != "" {
		t.Fatalf("second segment should be a space passthrough, got %+v", segs[1])
	}
	if segs[2].LineStart {
		t.Fatalf("third segment should not be LineStart, got %+v", segs[2])
	}
}

// TestEngineCapitalizationLines verifies that the first word of every line
// (after a newline) is capitalized.
func TestEngineCapitalizationLines(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)
	segs := eng.Transliterate("שָׁלוֹם\nשָׁלוֹם")
	if got := joinText(segs); got != "Shalom\nShalom" {
		t.Fatalf("got %q want %q", got, "Shalom\nShalom")
	}
}

// TestEnginePassthrough verifies non-Hebrew text is preserved verbatim,
// including a leading reference that should keep the following word lowercase.
func TestEnginePassthrough(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)
	segs := eng.Transliterate("Psalm 51: שָׁלוֹם")
	if got := joinText(segs); got != "Psalm 51: shalom" {
		t.Fatalf("got %q want %q", got, "Psalm 51: shalom")
	}
	if segs[0].Text != "Psalm 51: " {
		t.Fatalf("passthrough segment: got %q", segs[0].Text)
	}
	if segs[1].LineStart {
		t.Fatalf("word after non-space passthrough should not be LineStart")
	}
}

// TestEnginePrefixes covers the definite article and each inseparable
// preposition, plus a he-with-kamatz that must NOT be treated as an article.
func TestEnginePrefixes(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)

	cases := []struct {
		name   string
		hebrew string
		want   string
	}{
		{"definite article + dagesh body", "הַכֹּהֵן", "ha-kohen"},
		{"bet prefix", "בְּיוֹם", "be-yom"},
		{"kaf prefix", "כְּתִיב", "ke-tiv"},
		{"he+kamatz is not an article", "הָר", "har"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _, _ := eng.transliterateWord(tc.hebrew, false)
			if got != tc.want {
				t.Fatalf("hebrew=%q\n got %q\nwant %q", tc.hebrew, got, tc.want)
			}
		})
	}
}

// TestEngineDivineName covers both the Tetragrammaton and the double-yod
// euphemism, with and without nikud.
func TestEngineDivineName(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)
	for _, he := range []string{"יְהוָה", "יהוה", "יי"} {
		got, amb, _ := eng.transliterateWord(he, false)
		if got != "ADONAI" {
			t.Fatalf("hebrew=%q got %q want ADONAI", he, got)
		}
		if amb {
			t.Fatalf("divine name should not be flagged ambiguous, he=%q", he)
		}
	}
}

// TestEngineShvaAmbiguityFlags verifies the engine flags genuinely ambiguous
// shva cases (after short vowel / non-first) and does NOT flag the cases the
// rules fully specify (first-letter shva, shva after long vowel, shva under
// dagesh chazak).
func TestEngineShvaAmbiguityFlags(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)

	cases := []struct {
		name      string
		hebrew    string
		ambiguous bool
	}{
		{"first-letter shva is vocal (not flagged)", "שְׁמַע", false},
		{"shva after short vowel flagged (midbar)", "מִדְבָּר", true},
		{"shva after short vowel flagged (ach)", "אַךְ", true},
		{"shva after short vowel flagged (ahavta)", "וְאָהַבְתָּ", true},
		{"shva under dagesh chazak is vocal (not flagged)", "חַבְּרֵי", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, amb, _ := eng.transliterateWord(tc.hebrew, false)
			if amb != tc.ambiguous {
				t.Fatalf("hebrew=%q ambiguous: got %v want %v", tc.hebrew, amb, tc.ambiguous)
			}
		})
	}
}

// TestEngineShuruk covers vav+dagesh (shuruk) → u, distinct from consonantal v.
func TestEngineShuruk(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)
	got, _, _ := eng.transliterateWord("קוּם", false)
	if got != "kum" {
		t.Fatalf("got %q want %q", got, "kum")
	}
}

// TestEngineBareYodConsonant covers a bare yod acting as a consonant onset
// before a vowel (not a vowel carrier, not flagged).
func TestEngineBareYodConsonant(t *testing.T) {
	t.Parallel()

	eng := NewEngine(nil)
	got, amb, _ := eng.transliterateWord("מַיִם", false)
	if got != "mayim" {
		t.Fatalf("got %q want %q", got, "mayim")
	}
	if amb {
		t.Fatalf("mayim should not be flagged ambiguous")
	}
}

// joinText concatenates the Text field of every segment.
func joinText(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}