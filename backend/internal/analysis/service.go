package analysis

import (
	"context"
	"fmt"
	"obucon/internal/lang/ja"
	"obucon/internal/models"
	"strings"
)

type AnalysisResult struct {
	Tokens      []EnrichedToken `json:"tokens"`
	TotalTokens int             `json:"total_tokens"`
	Missing     []string        `json:"missing"`
}

type EnrichedToken struct {
	Surface       string `json:"surface"`
	Lemma         string `json:"lemma"`
	POS           string `json:"pos"`
	IsKnown       bool   `json:"is_known"`
	GradeLevel    *int   `json:"grade_level"`
	IsKatakana    bool   `json:"is_katakana"`
	IsRoman       bool   `json:"is_roman"`
	IsNonJapanese bool   `json:"is_non_japanese"`
	Meaning       string `json:"meaning"`
}

type Service struct {
	tokenizer *ja.Tokenizer
	repo      *Repository
}

func NewService(tokenizer *ja.Tokenizer, repo *Repository) *Service {
	return &Service{tokenizer: tokenizer, repo: repo}
}

type AddKnownWordResult struct {
	Lemma      string `json:"lemma"`
	GradeLevel *int   `json:"grade_level"`
	Status     string `json:"status"`
}

type UpdateKnownWordResult struct {
	Lemma      string `json:"lemma"`
	Meaning    string `json:"meaning"`
	GradeLevel int    `json:"grade_level"`
}

func (s *Service) AnalyzeText(ctx context.Context, userID uint, language, text string) (*AnalysisResult, error) {
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

	meanings, err := s.repo.GetDictionaryMeanings(ctx, language, lemmas)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dictionary meanings: %w", err)
	}

	enrichedTokens := make([]EnrichedToken, 0, len(tokens))
	missingSeen := make(map[string]struct{})
	missingSlice := make([]string, 0)

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

		isGrammarToken := strings.Contains(token.PartOfSpeech, "助詞") ||
			strings.Contains(token.PartOfSpeech, "助動詞") ||
			strings.Contains(token.PartOfSpeech, "記号") ||
			strings.Contains(token.PartOfSpeech, "動詞 接尾") ||
			strings.Contains(token.PartOfSpeech, "動詞 非自立")

		baseLemma := token.Lemma
		if before, ok := strings.CutSuffix(baseLemma, "さ"); ok {
			baseLemma = before
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

		if !isKnown && !token.IsNonJapanese {
			_, hasGrade := gradeLevels[token.Lemma]
			_, hasGradeBase := gradeLevels[baseLemma]
			if !hasGrade && !hasGradeBase {
				if _, seen := missingSeen[token.Lemma]; !seen {
					missingSeen[token.Lemma] = struct{}{}
					missingSlice = append(missingSlice, token.Lemma)
				}
			}
		}

		meaning, ok := meanings[token.Lemma]
		if !ok {
			meaning = meanings[baseLemma]
		}

		enrichedTokens = append(enrichedTokens, EnrichedToken{
			Surface:       token.Surface,
			Lemma:         token.Lemma,
			POS:           token.PartOfSpeech,
			IsKnown:       isKnown,
			GradeLevel:    gradeLevel,
			IsKatakana:    token.IsKatakana,
			IsRoman:       token.IsRoman,
			IsNonJapanese: token.IsNonJapanese,
			Meaning:       meaning,
		})
	}

	return &AnalysisResult{
		Tokens:      enrichedTokens,
		TotalTokens: len(enrichedTokens),
		Missing:     missingSlice,
	}, nil
}

func (s *Service) ListKnownVocabulary(ctx context.Context, userID uint, language string) ([]VocabEntry, error) {
	return s.repo.ListKnownWordsWithMeaning(ctx, userID, language)
}

func (s *Service) AddBulkKnownVocabulary(ctx context.Context, userID uint, language string, jlptLevel int) (int64, error) {
	return s.repo.BulkAddKnownWordsByJLPT(ctx, userID, language, jlptLevel)
}

func (s *Service) AddKnownWord(ctx context.Context, userID uint, language, lemma string) (*AddKnownWordResult, error) {
	cleanLemma := strings.TrimSpace(lemma)
	if cleanLemma == "" {
		return nil, fmt.Errorf("lemma cannot be empty")
	}

	word := &models.KnownWord{
		UserID:   userID,
		Language: language,
		Lemma:    cleanLemma,
		Status:   "known",
	}

	if err := s.repo.UpsertKnownWord(ctx, word); err != nil {
		return nil, fmt.Errorf("failed to store known word: %w", err)
	}

	return &AddKnownWordResult{
		Lemma:      word.Lemma,
		GradeLevel: word.GradeLevel,
		Status:     "known",
	}, nil
}

func (s *Service) UpdateKnownWord(ctx context.Context, userID uint, language, lemma, meaning string, jlptLevel int) (*UpdateKnownWordResult, error) {
	cleanLemma := strings.TrimSpace(lemma)
	if cleanLemma == "" {
		return nil, fmt.Errorf("lemma cannot be empty")
	}

	cleanMeaning := strings.TrimSpace(meaning)
	if cleanMeaning == "" {
		return nil, fmt.Errorf("meaning cannot be empty")
	}

	if jlptLevel < 1 || jlptLevel > 5 {
		return nil, fmt.Errorf("jlpt level must be between 1 and 5")
	}

	err := s.repo.UpdateKnownWord(ctx, userID, language, cleanLemma, cleanMeaning, jlptLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to update known word: %w", err)
	}

	return &UpdateKnownWordResult{
		Lemma:      cleanLemma,
		Meaning:    cleanMeaning,
		GradeLevel: jlptLevel,
	}, nil
}

func (s *Service) RemoveKnownWord(ctx context.Context, userID uint, language, lemma string) error {
	cleanLemma := strings.TrimSpace(lemma)
	if cleanLemma == "" {
		return fmt.Errorf("lemma cannot be empty")
	}

	err := s.repo.RemoveKnownWord(ctx, userID, language, cleanLemma)
	if err != nil {
		return fmt.Errorf("failed to remove known word: %w", err)
	}

	return nil
}

func (s *Service) GetReviewWords(ctx context.Context, language string, lemmas []string) ([]ReviewWord, error) {
	if len(lemmas) == 0 {
		return nil, nil
	}
	return s.repo.GetDictionaryEntries(ctx, language, lemmas)
}

func (s *Service) ListDictionary(ctx context.Context, language string) ([]DictionaryEntry, error) {
	return s.repo.ListDictionary(ctx, language)
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
