package transliteration

const DefaultRules = `
HEBREW TRANSLITERATION RULES (Modern Israeli / Academy of the Hebrew Language)**

Transliterate pointed Hebrew text (with nikud) into English characters following these rules. Reflect modern Israeli (Sephardic) pronunciation.

**Consonants**

|Hebrew|Transliteration|Notes|
|---|---|---|
|א aleph|(nothing)|always silent|
|ב with dagesh|b||
|ב without dagesh|v||
|ג|g|always hard, as in "get"|
|ד|d||
|ה|h|silent at end of word|
|ו|v|when consonantal|
|ז|z||
|ח|ch|as in Scottish "loch"|
|ט|t||
|י|y|when consonantal|
|כ / ך with dagesh|k||
|כ / ך without dagesh|kh|as in "Bach"|
|ל|l||
|מ / ם|m||
|נ / ן|n||
|ס|s||
|ע ayin|(nothing)|always silent in modern Hebrew|
|פ with dagesh|p||
|פ / ף without dagesh|f||
|צ / ץ|ts||
|ק|k||
|ר|r||
|שׁ shin dot|sh||
|שׂ sin dot|s||
|ת|t||

**Vowels (nikud)**

|Nikud|Name|Transliteration|Notes|
|---|---|---|---|
|ַ|patach|a||
|ָ|kamatz|a|kamatz katan → o (rare, context-dependent)|
|ֶ|segol|e||
|ֵ|tsere|e||
|ִ|chirik|i||
|ֹ|cholam|o||
|וֹ|cholam male|o||
|ֻ|kubutz|u||
|וּ|shuruk|u||
|ְ|shva|see notes|vocal shva → e; silent shva → nothing|
|ֲ|chataf patach|a||
|ֱ|chataf segol|e||
|ֳ|chataf kamatz|o||

**Key Rules**

1. **Shva**: A shva under the first letter of a word is always vocal (→ e). A shva after a long vowel is silent. A shva under a letter with dagesh chazak is vocal.
2. **Dagesh**: Dagesh chazak (in the middle of a word) doubles the consonant — represent as a single consonant (do not write "mm", "nn", etc.), since doubled consonants are not meaningful to English readers.
3. **Definite article**: ה at the start of a word with a patach (הַ) → "ha-". Use a hyphen to separate it: הַבַּיִת → ha-bayit.
4. **Vav as vowel vs. consonant**: וֹ and וּ are vowels (o, u). A vav with no nikud of its own, between vowels, is the consonant v.
5. **End-of-word ה**: Usually silent — do not transliterate (e.g. שָׂדֶה → sade). Exception: lexicalized liturgical forms where convention keeps the final h, e.g. סֶלָה → selah.
6. **Diacritics**: Do not use any diacritics or special characters. Use only standard ASCII letters.
7. **Word spacing**: Preserve word boundaries. Use a hyphen only for the definite article and inseparable prepositions (בְּ, לְ, כְּ, וְ → be-, le-, ke-, ve-).
8. **Kamatz Katan**: Default to "a" when uncertain, but recognize that it may sound like "o" in some contexts.
9. When encountering the Divine Name (י-ה-ו-ה or one of the abbreviations or euphemisms such as יי, or the Tetragrammaton represented by the letter ה alone), transliterate as "ADONAI".
10. Capitalize the first letter of each line or verse.
11. **Begadkefat hard/soft rule**: For ב/כ/פ, the dagesh determines hard vs. soft regardless of word position. With dagesh: ב=b, כ/ך=k, פ=p. Without dagesh: ב=v, כ/ך=kh, פ/ף=f.

**Examples** — apply the rules exactly as shown, with the exception that proper nouns may be transliterated according to common usage, and capitalization is appropriate at the first letter of line or verse.

|Hebrew|Output|Rule demonstrated|
|---|---|---|
|שְׁמַע|shema|leading shva → e; shin dot → sh|
|מִדְבָּר|midbar|silent shva closes a syllable (no vowel inserted)|
|שַׁבָּת|shabat|dagesh chazak → single consonant (no doubling)|
|הַבַּיִת|ha-bayit|definite article הַ → ha- with hyphen|
|שָׂדֶה|sade|silent final ה; sin dot → s|
|טוֹב|tov|cholam male (vav + ֹ) → o; bet without dagesh → v|
|אֹמֶר|omer|bare cholam (ֹ on a consonant, no vav) → o; segol → e|
|קֹדֶשׁ|kodesh|bare cholam → o; shin dot → sh|
|עֵץ|ets|tsere → e; silent ayin; final tsadi → ts|
|בְּיוֹם|be-yom|inseparable preposition בְּ → be- with hyphen; cholam → o|
|וְאָהַבְתָּ|ve-ahavta|vav shva → ve-; silent shva in בְ; kamatz → a|
|בֵיתֶךָ|veitekha|word-initial beit WITHOUT dagesh is v; final khaf without dagesh is kh|
|יְהַלְלוּךָ|yehalelukha|final khaf without dagesh is kh (not k)|
|סֶלָה|selah|lexicalized final ה is written as h|
|בָּשָׂר|basar|word-initial beit WITH dagesh → b (hard)|
|בָא|va|word-initial beit WITHOUT dagesh → v (soft)|
|אָבִיב|aviv|beit without dagesh after a vowel → v|
|כָּבוֹד|kavod|word-initial kaf WITH dagesh → k (hard)|
|כִי|khi|word-initial kaf WITHOUT dagesh → kh (soft)|
|אַךְ|akh|final kaf without dagesh → kh|
|יְהוָה|ADONAI|Divine Name → ADONAI|

Always use the rules above consistently when transliterating Hebrew text into English characters.
---
`
