package analysis

import (
	"context"
	"fmt"
	"obucon/internal/lang/ja"
)

type AnalysisService interface {
	AnalyzeText(ctx context.Context, userID uint, language, text string) (*AnalysisResult, error)
}

type AnalysisResult struct {
	Tokens      []ja.Token `json:"tokens"`
	TotalTokens int        `json:"total_tokens"`
}

type analysisService struct {
	tokenizer    *ja.Tokenizer
	analysisRepo AnalysisRepository
	tokenRepo    AnalysisTokenRepository
}

func NewAnalysisService(
	tokenizer *ja.Tokenizer,
	analysisRepo AnalysisRepository,
	tokenRepo AnalysisTokenRepository,
) AnalysisService {
	return &analysisService{
		tokenizer:    tokenizer,
		analysisRepo: analysisRepo,
		tokenRepo:    tokenRepo,
	}
}

// TODO:
// - Dictionary lookup for word definitions
// - Vocabulary comparison for known/unknown words
// - Coverage percentage calculation
func (s *analysisService) AnalyzeText(ctx context.Context, userID uint, language, text string) (*AnalysisResult, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	tokens, err := s.tokenizer.Tokenize(text)
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}

	return &AnalysisResult{
		Tokens:      tokens,
		TotalTokens: len(tokens),
	}, nil
}
