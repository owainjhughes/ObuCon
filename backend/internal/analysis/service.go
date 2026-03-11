package analysis

import (
	"context"
	"fmt"
	"obucon/internal/lang/ja"
	"strings"
)

type AnalysisResult struct {
	Tokens      []EnrichedToken `json:"tokens"`
	TotalTokens int             `json:"total_tokens"`
	Missing     []string        `json:"missing"`
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

	knownLemmas, err := s.repo.GetKnownLemmasWithDictionaryVariants(ctx, userID, language, lemmas)
	if err != nil {
		return nil, fmt.Errorf("failed to check known words: %w", err)
	}

	gradeLevels, err := s.repo.GetDictionaryGradeLevels(ctx, language, lemmas)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dictionary grade levels: %w", err)
	}

	enrichedTokens := make([]EnrichedToken, 0, len(tokens))
	missing := make(map[string]struct{})

	for _, token := range tokens {
		var gradeLevel *int
		if level, ok := gradeLevels[token.Lemma]; ok {
			value := level
			gradeLevel = &value
		} else if strings.HasSuffix(token.Lemma, "さ") {
			// Handle nominalized adjective forms like 美しさ by falling back to the base adjective.
			base := strings.TrimSuffix(token.Lemma, "さ")
			if level, ok := gradeLevels[base]; ok {
				value := level
				gradeLevel = &value
			}
		}

		isGrammarToken := strings.Contains(token.PartOfSpeech, "助詞") || strings.Contains(token.PartOfSpeech, "助動詞") || strings.Contains(token.PartOfSpeech, "記号") || strings.Contains(token.PartOfSpeech, "動詞 接尾") || strings.Contains(token.PartOfSpeech, "動詞 非自立")

		baseLemma := token.Lemma
		if strings.HasSuffix(baseLemma, "さ") {
			baseLemma = strings.TrimSuffix(baseLemma, "さ")
		}

		// For conjugated verbs, also allow matching against the dictionary root (e.g. 知る)
		rootLemma := baseLemma
		// Prefer stripping longer suffixes first to avoid over-stripping (e.g. られる -> る)
		if strings.HasSuffix(rootLemma, "られる") {
			rootLemma = strings.TrimSuffix(rootLemma, "られる")
		} else if strings.HasSuffix(rootLemma, "れる") {
			rootLemma = strings.TrimSuffix(rootLemma, "れる")
		}

		isKnown := isGrammarToken || knownLemmas[token.Lemma] || knownLemmas[token.Surface] || knownLemmas[baseLemma] || knownLemmas[rootLemma]

		if !isKnown {
			_, hasGrade := gradeLevels[token.Lemma]
			_, hasGradeBase := gradeLevels[baseLemma]
			if !hasGrade && !hasGradeBase {
				missing[token.Lemma] = struct{}{}
			}
		}

		enrichedTokens = append(enrichedTokens, EnrichedToken{
			Surface:    token.Surface,
			Lemma:      token.Lemma,
			POS:        token.PartOfSpeech,
			IsKnown:    isKnown,
			GradeLevel: gradeLevel,
		})
	}

	missingSlice := make([]string, 0, len(missing))
	for m := range missing {
		missingSlice = append(missingSlice, m)
	}

	return &AnalysisResult{
		Tokens:      enrichedTokens,
		TotalTokens: len(enrichedTokens),
		Missing:     missingSlice,
	}, nil
}

func (s *Service) ListKnownVocabulary(ctx context.Context, userID uint, language string) ([]VocabEntry, error) {
	fmt.Print("Analysis Service ListKnownVocabulary Function Reached\n")
	return s.repo.ListKnownWordsWithMeaning(ctx, userID, language)
}

func (s *Service) AddBulkKnownVocabulary(ctx context.Context, userID uint, language string, jlptLevel int) (int64, error) {
	fmt.Print("Analysis Service AddBulkKnownVocabulary Function Reached\n")
	return s.repo.BulkAddKnownWordsByJLPT(ctx, userID, language, jlptLevel)
}

func uniqueLemmas(tokens []ja.Token) []string {
	seen := make(map[string]struct{}, len(tokens)*2)
	lemmas := make([]string, 0, len(tokens)*2)

	add := func(s string) {
		if s == "" {
			return
		}
		if _, exists := seen[s]; exists {
			return
		}
		seen[s] = struct{}{}
		lemmas = append(lemmas, s)
	}

	for _, token := range tokens {
		add(token.Lemma)
		if token.Surface != token.Lemma {
			add(token.Surface)
		}
	}

	return lemmas
}
