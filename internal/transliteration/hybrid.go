package transliteration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// HybridService transliterates pointed Hebrew using a deterministic Engine for
// the bulk of the work and consulting an LLM Provider only for the words the
// engine flagged as genuinely ambiguous (e.g. a non-initial shva). With a nil
// provider it behaves as a pure, network-free engine.
type HybridService struct {
	engine   *Engine
	provider Provider // optional; nil → no LLM calls, engine output only
	rules    string
	logger   *slog.Logger
}

// NewHybridService builds a HybridService. rules are required (they are sent
// to the LLM as the authoritative spec for the ambiguous-word prompt). A nil
// provider is allowed and disables LLM consultation. A nil logger is replaced
// with a no-op handler.
func NewHybridService(provider Provider, rules string, logger *slog.Logger) (*HybridService, error) {
	if strings.TrimSpace(rules) == "" {
		return nil, errors.New("transliteration rules are required")
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &HybridService{
		engine:   NewEngine(logger),
		provider: provider,
		rules:    rules,
		logger:   logger,
	}, nil
}

// Transliterate runs the engine over the input, then — if there are flagged
// words and a provider is configured — asks the LLM for just those words and
// splices the answers back into the segment stream. The final string is the
// concatenation of all segment Text in order.
func (s *HybridService) Transliterate(ctx context.Context, req Request) (string, error) {
	input := strings.TrimSpace(req.Text)
	if input == "" {
		return "", errors.New("text is required")
	}

	segments := s.engine.Transliterate(input)

	// Collect unique ambiguous Hebrew words, preserving first-seen order so
	// repeated words reuse a single LLM answer.
	var ambiguous []string
	seen := map[string]bool{}
	for _, seg := range segments {
		if seg.Ambiguous && seg.Hebrew != "" && !seen[seg.Hebrew] {
			seen[seg.Hebrew] = true
			ambiguous = append(ambiguous, seg.Hebrew)
		}
	}

	s.logger.Debug("hybrid: engine pass complete",
		"segments", len(segments),
		"ambiguous_words", len(ambiguous))

	if len(ambiguous) == 0 {
		s.logger.Debug("hybrid: no ambiguous words; returning engine output")
		return joinSegments(segments), nil
	}

	if s.provider == nil {
		s.logger.Debug("hybrid: provider not configured; keeping engine best-guess",
			"ambiguous", ambiguous)
		return joinSegments(segments), nil
	}

	replacements, err := s.askLLM(ctx, ambiguous)
	if err != nil {
		// A provider/parsing failure is not fatal: fall back to the engine's
		// best-guess transliteration, which is already in the segments.
		s.logger.Debug("hybrid: LLM consultation failed; keeping engine best-guess",
			"error", err, "ambiguous", ambiguous)
		return joinSegments(segments), nil
	}

	for i := range segments {
		seg := &segments[i]
		if !seg.Ambiguous || seg.Hebrew == "" {
			continue
		}
		repl, ok := replacements[seg.Hebrew]
		if !ok {
			continue
		}
		repl = strings.TrimSpace(repl)
		if repl == "" {
			continue
		}
		// Normalize to the engine's capitalization convention: lowercase the
		// LLM's answer, then re-capitalize when the word starts a line/verse.
		repl = strings.ToLower(repl)
		if seg.LineStart {
			repl = capitalize(repl)
		}
		seg.Text = repl
	}

	return joinSegments(segments), nil
}

// askLLM sends the ambiguous Hebrew words to the provider as a JSON list and
// expects a JSON object mapping each Hebrew word to its transliteration.
func (s *HybridService) askLLM(ctx context.Context, words []string) (map[string]string, error) {
	system := buildSystemPrompt(s.rules) + "\n\n" +
		"For the words you are asked about, reply with ONLY a JSON object mapping each Hebrew word to its Latin transliteration. " +
		"Do not include prose, explanations, code fences, or any text outside the JSON object."

	wordsJSON, err := json.Marshal(words)
	if err != nil {
		return nil, fmt.Errorf("marshal words: %w", err)
	}
	user := fmt.Sprintf(`Transliterate each of these Hebrew words and return a JSON object of the form
{"<hebrew>": "<transliteration>"}.

Words:
%s`, string(wordsJSON))

	s.logger.Debug("hybrid: sending prompt to LLM",
		"system", system,
		"user", user,
		"word_count", len(words))

	resp, err := s.provider.Generate(ctx, system, user)
	if err != nil {
		return nil, err
	}

	s.logger.Debug("hybrid: received LLM response", "response", resp)

	return parseTransliterationMap(resp)
}

// parseTransliterationMap extracts the first JSON object from resp and
// decodes it as a map[string]string. It tolerates surrounding prose and
// ```json code fences.
func parseTransliterationMap(resp string) (map[string]string, error) {
	body := strings.TrimSpace(resp)
	body = strings.TrimPrefix(body, "```json")
	body = strings.TrimPrefix(body, "```")
	body = strings.TrimSuffix(body, "```")
	body = strings.TrimSpace(body)

	start := strings.Index(body, "{")
	end := strings.LastIndex(body, "}")
	if start < 0 || end < 0 || end <= start {
		return nil, errors.New("no JSON object found in LLM response")
	}
	obj := body[start : end+1]

	var m map[string]string
	if err := json.Unmarshal([]byte(obj), &m); err != nil {
		return nil, fmt.Errorf("parse JSON object: %w", err)
	}
	return m, nil
}

// joinSegments concatenates the Text field of every segment in order.
func joinSegments(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}