package analysis

import (
	"context"
	"fmt"
	"obucon/internal/lang/ja"
)

type AnalysisResult struct {
	Tokens      []ja.Token `json:"tokens"`
	TotalTokens int        `json:"total_tokens"`
}

type Service struct {
	tokenizer *ja.Tokenizer
}

func NewService(tokenizer *ja.Tokenizer) *Service {
	fmt.Print("Analysis Service NewService Function Reached\n")
	return &Service{tokenizer: tokenizer}
}

func (s *Service) AnalyzeText(ctx context.Context, userID uint, language, text string) (*AnalysisResult, error) {
	fmt.Print("Analysis Service AnalyzeText Function Reached\n")
	_ = ctx
	_ = userID
	_ = language

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
