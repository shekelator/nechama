package transliteration

import (
	"io"
	"log/slog"
	"strings"
	"unicode"
)

// Engine transliterates pointed Hebrew (with nikud) to Latin letters
// deterministically, following the rules encoded in this file. It handles the
// mechanical cases (consonant/vowel lookup, dagesh, final letters, cholam/
// shuruk, definite article, capitalization, explicit Divine Name forms) and
// flags genuinely ambiguous cases (e.g. a non-initial shva) for an LLM to
// review via the HybridService.
type Engine struct {
	logger *slog.Logger
}

// NewEngine returns an Engine that logs decisions at debug level. A nil logger
// is replaced with a no-op handler so the engine is safe to use without
// logging configured.
func NewEngine(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Engine{logger: logger}
}

// Segment is one piece of transliterated output. Passthrough segments (spaces,
// punctuation, Latin text, newlines) carry only Text; word segments also
// carry the original Hebrew and an ambiguity flag.
type Segment struct {
	// Text is the transliteration (for word segments) or the verbatim run of
	// non-Hebrew characters (for passthrough segments).
	Text string
	// Hebrew is the original Hebrew token (empty for passthrough segments).
	Hebrew string
	// Ambiguous is true when the engine flagged this word for LLM review. Text
	// still holds the engine's best-guess transliteration.
	Ambiguous bool
	// Reason describes why the word was flagged (empty when not ambiguous).
	Reason string
	// LineStart is true when this word begins a line of output; the hybrid
	// uses it to capitalize LLM replacements that land at line start.
	LineStart bool
}

// Transliterate splits text into a segment stream. Reassembling the segments
// in order (concatenating Text) yields the full transliteration with original
// spacing and line breaks preserved.
func (e *Engine) Transliterate(text string) []Segment {
	runes := []rune(text)
	var segments []Segment

	i := 0
	atLineStart := true
	for i < len(runes) {
		if isHebrewLetter(runes[i]) {
			j := i
			for j < len(runes) && (isHebrewLetter(runes[j]) || isHebrewMark(runes[j])) {
				j++
			}
			word := string(runes[i:j])
			out, amb, reason := e.transliterateWord(word, atLineStart)
			segments = append(segments, Segment{
				Text:      out,
				Hebrew:    word,
				Ambiguous: amb,
				Reason:    reason,
				LineStart: atLineStart,
			})
			atLineStart = false
			i = j
		} else {
			j := i
			for j < len(runes) && !isHebrewLetter(runes[j]) {
				j++
			}
			pass := string(runes[i:j])
			segments = append(segments, Segment{Text: pass})
			for _, r := range pass {
				if r == '\n' {
					atLineStart = true
				} else if !unicode.IsSpace(r) {
					atLineStart = false
				}
			}
			i = j
		}
	}
	return segments
}

// transliterateWord handles a single Hebrew word token and returns its
// transliteration, whether it was flagged ambiguous, and a reason if so.
func (e *Engine) transliterateWord(word string, lineStart bool) (string, bool, string) {
	slots := parseSlots(word)
	if len(slots) == 0 {
		return "", false, ""
	}

	if isDivineName(slots) {
		e.logger.Debug("engine: divine name", "hebrew", word, "output", "ADONAI")
		return "ADONAI", false, ""
	}

	var out strings.Builder
	ambiguous := false
	reason := ""

	isFirst := true
	prevClass := "none"   // last vowel class: short|long|kamatz|none
	prevVowelStr := ""    // last REAL vowel emitted (a|e|i|o|u), for yod-carrier detection

	i := 0
	// Optional leading prefix: definite article (הַ → "ha-") or inseparable
	// prepositions (בְּ/לְ/כְּ/וְ → be-/le-/ke-/ve-).
	if prefix, ok := detectPrefix(slots); ok && len(slots) > 1 {
		out.WriteString(prefix)
		isFirst = true
		prevClass = "short"
		prevVowelStr = ""
		i = 1
	}

	for i < len(slots) {
		slot := slots[i]
		isLast := i == len(slots)-1
		letter := slot.letter

		// Vav: vowel (cholam/shuruk) or consonant with a following vowel.
		if letter == 'ו' {
			switch {
			case slot.vowel == rCholam || slot.vowel == rCholamHaser:
				out.WriteString("o")
				prevClass, prevVowelStr = "long", "o"
			case slot.hasDagesh && slot.vowel == 0 && !slot.hasShva:
				out.WriteString("u")
				prevClass, prevVowelStr = "long", "u"
			default:
				emit, amb, rsn, class, vstr := emitConsonant("v", slot, isFirst, prevClass)
				out.WriteString(emit)
				ambiguous, reason = mergeAmb(ambiguous, reason, amb, rsn)
				prevClass, prevVowelStr = class, vstr
			}
			isFirst = false
			i++
			continue
		}

		// Bare yod (no marks): vowel carrier after i/e, otherwise consonant.
		if letter == 'י' && slot.vowel == 0 && !slot.hasShva && !slot.hasDagesh {
			if prevVowelStr == "i" || prevVowelStr == "e" {
				// chirik/segol/tsere male: yod is a vowel carrier, emit nothing.
				e.logger.Debug("engine: yod carrier", "hebrew", word)
				i++
				continue
			}
			if isLast || !nextHasVowel(slots, i+1) {
				out.WriteString("y")
				ambiguous, reason = mergeAmb(ambiguous, reason, true, "bare yod consonant")
				prevClass, prevVowelStr = "none", ""
			} else {
				out.WriteString("y")
				prevClass, prevVowelStr = "none", ""
			}
			isFirst = false
			i++
			continue
		}

		// Silent final he (no vowel, no shva): drop.
		if isLast && letter == 'ה' && slot.vowel == 0 && !slot.hasShva {
			e.logger.Debug("engine: silent final he", "hebrew", word)
			isFirst = false
			i++
			continue
		}

		cons := consonantMap(letter, slot.hasDagesh, slot.hasShinDot, slot.hasSinDot)
		emit, amb, rsn, class, vstr := emitConsonant(cons, slot, isFirst, prevClass)
		out.WriteString(emit)
		ambiguous, reason = mergeAmb(ambiguous, reason, amb, rsn)
		prevClass, prevVowelStr = class, vstr
		isFirst = false
		i++
	}

	result := out.String()
	if lineStart {
		result = capitalize(result)
	}
	e.logger.Debug("engine: word", "hebrew", word, "output", result, "ambiguous", ambiguous, "reason", reason)
	return result, ambiguous, reason
}

// emitConsonant produces the consonant + its vowel, resolving shva. Returns
// the emitted string, ambiguity flag/reason, and updated vowel class/string.
func emitConsonant(cons string, slot slot, isFirst bool, prevClass string) (string, bool, string, string, string) {
	if slot.hasShva {
		str, amb, rsn, class := resolveShva(slot, isFirst, prevClass)
		return cons + str, amb, rsn, class, ""
	}
	vstr, class := vowelOf(slot.vowel)
	return cons + vstr, false, "", class, vstr
}

// resolveShva decides a shva (U+05B0). Per the rules: a shva under the first
// letter of a word is vocal (→ e); a shva after a long vowel is silent; a
// shva under a dagesh chazak is vocal. Everything else (after a short vowel,
// kamatz, another shva, or no vowel) is genuinely ambiguous — best-guess
// silent, and flagged for LLM review.
func resolveShva(slot slot, isFirst bool, prevClass string) (string, bool, string, string) {
	if isFirst || isDageshChazak(slot, isFirst) {
		return "e", false, "", "short"
	}
	if prevClass == "long" {
		return "", false, "", "none"
	}
	return "", true, "shva not first / after " + prevClass, "none"
}

// mergeAmb keeps the first ambiguity reason for logging.
func mergeAmb(amb bool, reason, newAmb bool, newReason string) (bool, string) {
	if !amb && newAmb {
		return true, newReason
	}
	return amb, reason
}

// capitalize uppercases the first rune of s (for line/verse-initial words).
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToTitle(r[0])
	return string(r)
}

// nextHasVowel reports whether the slot at index k provides a vowel on its own
// (a nikud vowel, or a vav carrying cholam/shuruk). Used to decide whether a
// bare yod is a consonant onset.
func nextHasVowel(slots []slot, k int) bool {
	if k < 0 || k >= len(slots) {
		return false
	}
	s := slots[k]
	if s.letter == 'ו' && (s.vowel == rCholam || s.vowel == rCholamHaser || (s.hasDagesh && s.vowel == 0)) {
		return true
	}
	return s.vowel != 0 || s.hasShva
}

// detectPrefix returns the leading prefix string (with trailing hyphen) and ok
// when the word starts with a definite article or inseparable preposition
// that has a following body.
func detectPrefix(slots []slot) (string, bool) {
	if len(slots) < 2 {
		return "", false
	}
	s := slots[0]
	switch s.letter {
	case 'ה':
		if s.vowel == rPatach && !s.hasDagesh {
			return "ha-", true
		}
	case 'ב':
		if s.hasShva {
			return "be-", true
		}
	case 'ל':
		if s.hasShva {
			return "le-", true
		}
	case 'כ', 'ך':
		if s.hasShva {
			return "ke-", true
		}
	case 'ו':
		if s.hasShva {
			return "ve-", true
		}
	}
	return "", false
}

// isDivineName reports whether the word is the Tetragrammaton (י-ה-ו-ה) or the
// double-yod euphemism (יי), with or without nikud.
func isDivineName(slots []slot) bool {
	letters := make([]rune, 0, len(slots))
	for _, s := range slots {
		if s.letter != 0 {
			letters = append(letters, s.letter)
		}
	}
	switch string(letters) {
	case "יהוה", "יי":
		return true
	}
	return false
}

// slot is a single Hebrew consonant letter together with its combining marks.
type slot struct {
	letter     rune
	marks       []rune
	vowel       rune // the vowel nikud (0 if none)
	hasDagesh   bool
	hasShva     bool
	hasShinDot  bool
	hasSinDot   bool
}

// parseSlots splits a Hebrew word token into letter slots with their attached
// combining marks.
func parseSlots(word string) []slot {
	var slots []slot
	var cur *slot
	for _, r := range word {
		if isHebrewLetter(r) {
			slots = append(slots, slot{letter: r})
			cur = &slots[len(slots)-1]
		} else if isHebrewMark(r) && cur != nil {
			cur.marks = append(cur.marks, r)
		}
	}
	for i := range slots {
		deriveMarks(&slots[i])
	}
	return slots
}

// deriveMarks classifies a slot's combining marks into vowel/dagesh/etc.
func deriveMarks(s *slot) {
	for _, r := range s.marks {
		switch r {
		case rDagesh:
			s.hasDagesh = true
		case rShinDot:
			s.hasShinDot = true
		case rSinDot:
			s.hasSinDot = true
		case rShva:
			s.hasShva = true
			s.vowel = rShva
		default:
			if isVowelMark(r) && s.vowel == 0 {
				s.vowel = r
			}
		}
	}
}

// isDageshChazak reports whether a slot's dagesh is a chazak (gemination)
// rather than a lene (begadkefat hardening at the start of a word). A shva
// under a dagesh chazak is vocal.
func isDageshChazak(s slot, isFirst bool) bool {
	if !s.hasDagesh {
		return false
	}
	if !isBegadkefat(s.letter) {
		return true
	}
	return !isFirst
}

// vowelOf maps a vowel nikud rune to its Latin output and class (short/long/
// kamatz/none). Shva is handled separately by resolveShva.
func vowelOf(r rune) (string, string) {
	switch r {
	case rHatafSegol:
		return "e", "short"
	case rHatafPatach:
		return "a", "short"
	case rHatafKamatz:
		return "o", "short"
	case rChirik:
		return "i", "short"
	case rTsere:
		return "e", "long"
	case rSegol:
		return "e", "short"
	case rPatach:
		return "a", "short"
	case rKamatz:
		return "a", "kamatz"
	case rCholam, rCholamHaser:
		return "o", "long"
	case rKubutz:
		return "u", "short"
	default:
		return "", "none"
	}
}

// consonantMap maps a Hebrew consonant letter to its Latin form. Dagesh only
// changes the output for begadkefat (ב/כ/פ → hard b/k/p); dagesh chazak on
// other letters is a single consonant per rule 2 (no doubling).
func consonantMap(letter rune, hasDagesh, hasShinDot, hasSinDot bool) string {
	switch letter {
	case 'א', 'ע':
		return ""
	case 'ב':
		if hasDagesh {
			return "b"
		}
		return "v"
	case 'ג':
		return "g"
	case 'ד':
		return "d"
	case 'ה':
		return "h"
	case 'ו':
		return "v"
	case 'ז':
		return "z"
	case 'ח':
		return "ch"
	case 'ט':
		return "t"
	case 'י':
		return "y"
	case 'כ', 'ך':
		if hasDagesh {
			return "k"
		}
		return "ch"
	case 'ל':
		return "l"
	case 'מ', 'ם':
		return "m"
	case 'נ', 'ן':
		return "n"
	case 'ס':
		return "s"
	case 'פ', 'ף':
		if hasDagesh {
			return "p"
		}
		return "f"
	case 'צ', 'ץ':
		return "ts"
	case 'ק':
		return "k"
	case 'ר':
		return "r"
	case 'ש':
		if hasSinDot {
			return "s"
		}
		return "sh" // shin dot or unspecified → shin
	case 'ת':
		return "t"
	}
	return string(letter)
}

func isBegadkefat(r rune) bool {
	switch r {
	case 'ב', 'ג', 'ד', 'כ', 'ך', 'פ', 'ף', 'ת':
		return true
	}
	return false
}

// Nikud code points.
const (
	rShva        = 0x05B0
	rHatafSegol  = 0x05B1
	rHatafPatach = 0x05B2
	rHatafKamatz = 0x05B3
	rChirik      = 0x05B4
	rTsere       = 0x05B5
	rSegol       = 0x05B6
	rPatach      = 0x05B7
	rKamatz      = 0x05B8
	rCholam      = 0x05B9
	rCholamHaser = 0x05BA
	rKubutz      = 0x05BB
	rDagesh      = 0x05BC
	rShinDot     = 0x05C1
	rSinDot      = 0x05C2
)

func isVowelMark(r rune) bool {
	switch r {
	case rHatafSegol, rHatafPatach, rHatafKamatz,
		rChirik, rTsere, rSegol, rPatach, rKamatz,
		rCholam, rCholamHaser, rKubutz:
		return true
	}
	return false
}

func isHebrewLetter(r rune) bool {
	return unicode.Is(unicode.Hebrew, r) && unicode.IsLetter(r)
}

func isHebrewMark(r rune) bool {
	return unicode.Is(unicode.Hebrew, r) && unicode.IsMark(r)
}