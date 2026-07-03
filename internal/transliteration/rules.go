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
|כ / ך without dagesh|ch|as in "loch"|
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
5. **End-of-word ה**: Silent — do not transliterate. e.g. שָׂדֶה → sade.
6. **Diacritics**: Do not use any diacritics or special characters. Use only standard ASCII letters.
7. **Word spacing**: Preserve word boundaries. Use a hyphen only for the definite article and inseparable prepositions (בְּ, לְ, כְּ, וְ → be-, le-, ke-, ve-).

---

This should give a model enough to be consistent. The main ambiguity you may still encounter is **kamatz katan** (which sounds like "o" not "a") — that's genuinely hard to resolve without grammatical parsing, so you may want to instruct the model to default to "a" and flag uncertain cases if precision matters.

`
