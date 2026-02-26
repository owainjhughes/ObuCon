package ja

import (
	"encoding/json"
	"fmt"

	"github.com/ikawaha/kagome/v2/tokenizer"
)

type TokenAnalysis struct {
	Surface string   `json:"surface"`
	POS     []string `json:"pos"`
	Base    string   `json:"base,omitempty"`
}

type AnalysisResult struct {
	Text   string          `json:"text"`
	Tokens []TokenAnalysis `json:"tokens"`
	Count  int             `json:"token_count"`
}

func AnalyzeText(t *tokenizer.Tokenizer, text string) *AnalysisResult {
	tokens := t.Tokenize(text)
	analysis := make([]TokenAnalysis, 0, len(tokens))

	for _, token := range tokens {
		ta := TokenAnalysis{
			Surface: token.Surface,
			POS:     token.POS(),
		}

		if len(token.POS()) > 6 && token.POS()[6] != "*" {
			ta.Base = token.POS()[6]
		}

		analysis = append(analysis, ta)
	}

	return &AnalysisResult{
		Text:   text,
		Tokens: analysis,
		Count:  len(analysis),
	}
}

func (ar *AnalysisResult) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(ar, "", "  ")
	return string(data), err
}

func (ar *AnalysisResult) FormatCSV() string {
	csv := "Surface,POS\n"
	for _, token := range ar.Tokens {
		pos := "-"
		if len(token.POS) > 0 {
			pos = token.POS[0]
		}
		csv += fmt.Sprintf("%s,%s\n", token.Surface, pos)
	}
	return csv
}
