package transliteration

import (
	"strings"
	"testing"
)

func TestDefaultRulesIncludeSoftKafAndBetGuidance(t *testing.T) {
	t.Parallel()

	checks := []string{
		"|כ / ך without dagesh|kh|",
		"|כִי|khi|",
		"|אַךְ|akh|",
		"|בֵיתֶךָ|veitekha|",
		"|יְהַלְלוּךָ|yehalelukha|",
		"|סֶלָה|selah|",
		"|מֵעַל|me'al|medial ayin marks a syllable break with an apostrophe|",
		"|רוּחַ|ruach|furtive patach under final ח is written before the consonant|",
		"|חוֹל|chol|cholam must keep the o; never drop the vowel|",
		"Cholam is never dropped",
		"Medial aleph/ayin",
		"Use a hyphen only where Hebrew itself uses maqaf",
	}

	for _, check := range checks {
		if !strings.Contains(DefaultRules, check) {
			t.Fatalf("expected DefaultRules to contain %q", check)
		}
	}
}
