package analysis

import (
	"context"
	"fmt"
	"obucon/internal/lang/ja"
)

type AnalysisResult struct {
	Tokens      []EnrichedToken `json:"tokens"`
	TotalTokens int             `json:"total_tokens"`
}

type EnrichedToken struct {
	Surface    string `json:"surface"`
	Lemma      string `json:"lemma"`
	POS        string `json:"pos"`
	IsKnown    bool   `json:"is_known"`
	GradeLevel *int   `json:"grade_level"`
}

type Service struct {
	tokenizer *ja.Tokenizer
	repo      *Repository
}

func NewService(tokenizer *ja.Tokenizer, repo *Repository) *Service {
	fmt.Print("Analysis Service NewService Function Reached\n")
	return &Service{tokenizer: tokenizer, repo: repo}
}

func (s *Service) AnalyzeText(ctx context.Context, userID uint, language, text string) (*AnalysisResult, error) {
	fmt.Print("Analysis Service AnalyzeText Function Reached\n")
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	tokens, err := s.tokenizer.Tokenize(text)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	lemmas := uniqueLemmas(tokens)

	knownLemmas, err := s.repo.GetKnownLemmas(ctx, userID, language, lemmas)
	if err != nil {
		return nil, fmt.Errorf("failed to check known words: %w", err)
	}

	gradeLevels, err := s.repo.GetDictionaryGradeLevels(ctx, language, lemmas)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dictionary grade levels: %w", err)
	}

	enrichedTokens := make([]EnrichedToken, 0, len(tokens))
	for _, token := range tokens {
		var gradeLevel *int
		if level, ok := gradeLevels[token.Lemma]; ok {
			value := level
			gradeLevel = &value
		}

		enrichedTokens = append(enrichedTokens, EnrichedToken{
			Surface:    token.Surface,
			Lemma:      token.Lemma,
			POS:        token.PartOfSpeech,
			IsKnown:    knownLemmas[token.Lemma],
			GradeLevel: gradeLevel,
		})
	}

	return &AnalysisResult{
		Tokens:      enrichedTokens,
		TotalTokens: len(enrichedTokens),
	}, nil
}

func uniqueLemmas(tokens []ja.Token) []string {
	seen := make(map[string]struct{}, len(tokens))
	lemmas := make([]string, 0, len(tokens))

	for _, token := range tokens {
		if token.Lemma == "" {
			continue
		}

		if _, exists := seen[token.Lemma]; exists {
			continue
		}

		seen[token.Lemma] = struct{}{}
		lemmas = append(lemmas, token.Lemma)
	}

	return lemmas
}
