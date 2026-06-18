package sefaria

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClientFetchSourceText(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v3/texts/Genesis 1:1"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}

		assertQueryContains(t, r.URL.Query(), "version", "source")
		assertQueryContains(t, r.URL.Query(), "return_format", "text_only")

		writeJSON(t, w, map[string]any{
			"ref":   "Genesis 1:1",
			"heRef": "בראשית א׳:א׳",
			"versions": []map[string]any{
				{
					"versionTitle":       "Miqra according to the Masorah",
					"shortVersionTitle":  "MAM",
					"languageFamilyName": "hebrew",
					"actualLanguage":     "he",
					"direction":          "rtl",
					"isSource":           true,
					"text":               "בְּ֑רֵֽאשִׁית&nbsp;&thinsp;׃",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	text, err := client.FetchText(context.Background(), FetchRequest{Ref: "Genesis 1:1", Language: LanguageSource})
	if err != nil {
		t.Fatalf("FetchText() error = %v", err)
	}

	if got, want := text.Text, "בְּרֵֽאשִׁית\u00A0\u2009׃"; got != want {
		t.Fatalf("unexpected text: got %q want %q", got, want)
	}
	if !text.IsSource {
		t.Fatal("expected source text")
	}
}

func TestClientFetchSpecificEnglishTranslation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertQueryContains(t, r.URL.Query(), "version", "english|THE JPS TANAKH: Gender-Sensitive Edition")
		writeJSON(t, w, map[string]any{
			"ref": "Genesis 1:1",
			"versions": []map[string]any{
				{
					"versionTitle":       "THE JPS TANAKH: Gender-Sensitive Edition",
					"shortVersionTitle":  "Revised JPS, 2023",
					"languageFamilyName": "english",
					"actualLanguage":     "en",
					"direction":          "ltr",
					"isSource":           false,
					"text":               "When God began&nbsp;to create heaven and earth&mdash;",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	text, err := client.FetchText(context.Background(), FetchRequest{
		Ref:              "Genesis 1:1",
		Language:         LanguageEnglish,
		TranslationTitle: "THE JPS TANAKH: Gender-Sensitive Edition",
	})
	if err != nil {
		t.Fatalf("FetchText() error = %v", err)
	}

	if got, want := text.ShortVersionTitle, "Revised JPS, 2023"; got != want {
		t.Fatalf("unexpected short title: got %q want %q", got, want)
	}
	if got, want := text.Text, "When God began\u00A0to create heaven and earth—"; got != want {
		t.Fatalf("unexpected text: got %q want %q", got, want)
	}
}

func TestClientFormatsNestedText(t *testing.T) {
	t.Parallel()

	value, err := json.Marshal([][]string{
		{"First comment", "Second comment"},
		{},
		{"Third comment"},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	text, err := formatTextValue(value)
	if err != nil {
		t.Fatalf("formatTextValue() error = %v", err)
	}

	if got, want := text, "First comment\nSecond comment\n\nThird comment"; got != want {
		t.Fatalf("unexpected flattened text: got %q want %q", got, want)
	}
}

func TestClientFormatsNestedTextWithNullEntries(t *testing.T) {
	t.Parallel()

	value := json.RawMessage(`[["First comment", null, "Second comment"], null, ["Third comment"]]`)
	text, err := formatTextValue(value)
	if err != nil {
		t.Fatalf("formatTextValue() error = %v", err)
	}

	if got, want := text, "First comment\nSecond comment\n\nThird comment"; got != want {
		t.Fatalf("unexpected flattened text: got %q want %q", got, want)
	}
}

func TestStripCantillationPreservesNekudotMetegAndSofPasuq(t *testing.T) {
	t.Parallel()

	input := "בְ֑רָֽא׃"
	if got, want := stripCantillation(input), "בְרָֽא׃"; got != want {
		t.Fatalf("unexpected normalized Hebrew: got %q want %q", got, want)
	}
}

func TestClientListsEnglishVersionsSortedByPriority(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertQueryContains(t, r.URL.Query(), "version", "english|all")
		writeJSON(t, w, map[string]any{
			"versions": []map[string]any{
				{
					"versionTitle":       "Version B",
					"shortVersionTitle":  "B",
					"languageFamilyName": "english",
					"priority":           1.0,
					"text":               "B",
				},
				{
					"versionTitle":       "Version A",
					"shortVersionTitle":  "A",
					"languageFamilyName": "english",
					"priority":           5.0,
					"text":               "A",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	versions, err := client.ListEnglishVersions(context.Background(), "Genesis 1:1")
	if err != nil {
		t.Fatalf("ListEnglishVersions() error = %v", err)
	}

	if got, want := versions[0].VersionTitle, "Version A"; got != want {
		t.Fatalf("unexpected first version: got %q want %q", got, want)
	}
}

func TestMatchTranslationMatchesShortOrFullTitle(t *testing.T) {
	t.Parallel()

	versions := []VersionChoice{
		{
			VersionTitle:      "THE JPS TANAKH: Gender-Sensitive Edition",
			ShortVersionTitle: "Revised JPS, 2023",
		},
	}

	match, err := MatchTranslation(versions, "revised   jps, 2023")
	if err != nil {
		t.Fatalf("MatchTranslation() error = %v", err)
	}

	if got, want := match.VersionTitle, "THE JPS TANAKH: Gender-Sensitive Edition"; got != want {
		t.Fatalf("unexpected version title: got %q want %q", got, want)
	}
}

func TestClientReturnsHelpfulErrorWhenNoVersionsExist(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"ref":      "Missing Ref",
			"versions": []map[string]any{},
			"warnings": []string{"Ref could not be resolved"},
		})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	_, err := client.FetchText(context.Background(), FetchRequest{Ref: "Missing Ref", Language: LanguageSource})
	if err == nil || !strings.Contains(err.Error(), "Ref could not be resolved") {
		t.Fatalf("expected warning in error, got %v", err)
	}
}

func assertQueryContains(t *testing.T, values url.Values, key, want string) {
	t.Helper()

	got := values[key]
	if !strings.Contains(strings.Join(got, ","), want) {
		t.Fatalf("expected query %q to contain %q, got %v", key, want, got)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
