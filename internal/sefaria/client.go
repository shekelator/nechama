package sefaria

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const defaultBaseURL = "https://www.sefaria.org"

var (
	ErrNoText                = errors.New("no text available for the requested reference")
	ErrNoEnglishTranslations = errors.New("no English translations available for the requested reference")
)

type Language string

const (
	LanguageSource  Language = "source"
	LanguageEnglish Language = "english"
)

type FetchRequest struct {
	Ref              string
	Language         Language
	TranslationTitle string
}

type Text struct {
	Ref               string
	HeRef             string
	Text              string
	VersionTitle      string
	ShortVersionTitle string
	LanguageFamily    string
	ActualLanguage    string
	Direction         string
	IsSource          bool
}

type VersionChoice struct {
	VersionTitle      string
	ShortVersionTitle string
	LanguageFamily    string
	Priority          float64
}

func (v VersionChoice) DisplayTitle() string {
	if v.ShortVersionTitle == "" || v.ShortVersionTitle == v.VersionTitle {
		return v.VersionTitle
	}

	return fmt.Sprintf("%s — %s", v.ShortVersionTitle, v.VersionTitle)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

type ClientOption func(*Client)

func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) {
		if baseURL != "" {
			client.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithUserAgent(userAgent string) ClientOption {
	return func(client *Client) {
		if userAgent != "" {
			client.userAgent = userAgent
		}
	}
}

func NewClient(options ...ClientOption) *Client {
	client := &Client{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
		userAgent:  "nechama/dev",
	}

	for _, option := range options {
		option(client)
	}

	return client
}

func (c *Client) FetchText(ctx context.Context, req FetchRequest) (Text, error) {
	if strings.TrimSpace(req.Ref) == "" {
		return Text{}, errors.New("reference is required")
	}

	selector, err := languageSelector(req)
	if err != nil {
		return Text{}, err
	}

	endpoint := fmt.Sprintf("%s/api/v3/texts/%s", c.baseURL, url.PathEscape(strings.TrimSpace(req.Ref)))
	query := url.Values{}
	query.Add("version", selector)
	query.Add("return_format", "text_only")

	var payload textResponse
	if err := c.getJSON(ctx, endpoint, query, &payload); err != nil {
		return Text{}, err
	}

	if len(payload.Versions) == 0 {
		return Text{}, buildNoTextError(req.Ref, payload.Warnings)
	}

	version := payload.Versions[0]
	text, err := formatTextValue(version.Text)
	if err != nil {
		return Text{}, err
	}

	return Text{
		Ref:               payload.Ref,
		HeRef:             payload.HeRef,
		Text:              text,
		VersionTitle:      version.VersionTitle,
		ShortVersionTitle: version.ShortVersionTitle,
		LanguageFamily:    version.LanguageFamilyName,
		ActualLanguage:    version.ActualLanguage,
		Direction:         version.Direction,
		IsSource:          version.IsSource,
	}, nil
}

func (c *Client) ListEnglishVersions(ctx context.Context, ref string) ([]VersionChoice, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, errors.New("reference is required")
	}

	endpoint := fmt.Sprintf("%s/api/v3/texts/%s", c.baseURL, url.PathEscape(strings.TrimSpace(ref)))
	query := url.Values{}
	query.Add("version", "english|all")
	query.Add("return_format", "text_only")

	var payload textResponse
	if err := c.getJSON(ctx, endpoint, query, &payload); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	versions := make([]VersionChoice, 0, len(payload.Versions))
	for _, version := range payload.Versions {
		if version.LanguageFamilyName != string(LanguageEnglish) {
			continue
		}
		if _, ok := seen[version.VersionTitle]; ok {
			continue
		}
		seen[version.VersionTitle] = struct{}{}
		versions = append(versions, VersionChoice{
			VersionTitle:      version.VersionTitle,
			ShortVersionTitle: version.ShortVersionTitle,
			LanguageFamily:    version.LanguageFamilyName,
			Priority:          float64(version.Priority),
		})
	}

	sort.SliceStable(versions, func(i, j int) bool {
		return versions[i].Priority > versions[j].Priority
	})

	if len(versions) == 0 {
		return nil, ErrNoEnglishTranslations
	}

	return versions, nil
}

func MatchTranslation(versions []VersionChoice, requested string) (VersionChoice, error) {
	if len(versions) == 0 {
		return VersionChoice{}, ErrNoEnglishTranslations
	}

	needle := normalizeLabel(requested)
	if needle == "" {
		return VersionChoice{}, errors.New("translation name is required")
	}

	var matches []VersionChoice
	for _, version := range versions {
		candidates := []string{version.VersionTitle, version.ShortVersionTitle}
		if slices.ContainsFunc(candidates, func(candidate string) bool {
			return normalizeLabel(candidate) == needle
		}) {
			matches = append(matches, version)
		}
	}

	switch len(matches) {
	case 0:
		return VersionChoice{}, fmt.Errorf("translation %q was not found", requested)
	case 1:
		return matches[0], nil
	default:
		titles := make([]string, 0, len(matches))
		for _, match := range matches {
			titles = append(titles, match.DisplayTitle())
		}
		return VersionChoice{}, fmt.Errorf("translation %q is ambiguous; matches: %s", requested, strings.Join(titles, ", "))
	}
}

func normalizeLabel(label string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(label)))
	return strings.Join(fields, " ")
}

func languageSelector(req FetchRequest) (string, error) {
	if req.TranslationTitle != "" {
		return fmt.Sprintf("english|%s", req.TranslationTitle), nil
	}

	switch req.Language {
	case "", LanguageSource:
		return "source", nil
	case LanguageEnglish:
		return "english", nil
	default:
		return "", fmt.Errorf("unsupported language %q", req.Language)
	}
}

func formatTextValue(raw json.RawMessage) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}

	formatted, err := flattenValue(value)
	if err != nil {
		return "", err
	}

	return formatted.text, nil
}

type flattenedText struct {
	text   string
	nested bool
}

func flattenValue(value any) (flattenedText, error) {
	switch typed := value.(type) {
	case nil:
		return flattenedText{}, nil
	case string:
		return flattenedText{text: strings.TrimSpace(typed)}, nil
	case []any:
		parts := make([]string, 0, len(typed))
		childNested := false
		for _, item := range typed {
			flattened, err := flattenValue(item)
			if err != nil {
				return flattenedText{}, err
			}
			if flattened.text != "" {
				parts = append(parts, flattened.text)
			}
			childNested = childNested || flattened.nested
		}

		separator := "\n"
		if childNested {
			separator = "\n\n"
		}

		return flattenedText{
			text:   strings.Join(parts, separator),
			nested: true,
		}, nil
	default:
		return flattenedText{}, fmt.Errorf("unsupported text payload type %T", value)
	}
}

func buildNoTextError(ref string, warnings []string) error {
	if len(warnings) == 0 {
		return fmt.Errorf("%w: %s", ErrNoText, ref)
	}

	return fmt.Errorf("%w: %s (%s)", ErrNoText, ref, strings.Join(warnings, "; "))
}

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", c.userAgent)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return fmt.Errorf("Sefaria API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(response.Body).Decode(target)
}

type textResponse struct {
	Ref      string            `json:"ref"`
	HeRef    string            `json:"heRef"`
	Versions []responseVersion `json:"versions"`
	Warnings []string          `json:"warnings"`
}

type responseVersion struct {
	VersionTitle       string          `json:"versionTitle"`
	ShortVersionTitle  string          `json:"shortVersionTitle"`
	LanguageFamilyName string          `json:"languageFamilyName"`
	ActualLanguage     string          `json:"actualLanguage"`
	Direction          string          `json:"direction"`
	IsSource           bool            `json:"isSource"`
	Priority           flexibleFloat   `json:"priority"`
	Text               json.RawMessage `json:"text"`
}

type flexibleFloat float64

func (f *flexibleFloat) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" || trimmed == `""` {
		*f = 0
		return nil
	}

	var number float64
	if err := json.Unmarshal(data, &number); err == nil {
		*f = flexibleFloat(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		*f = 0
		return nil
	}

	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return err
	}

	*f = flexibleFloat(parsed)
	return nil
}
