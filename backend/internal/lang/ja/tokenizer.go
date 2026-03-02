package ja

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

type Token struct {
	Surface      string `json:"surface"`
	Lemma        string `json:"lemma"`
	PartOfSpeech string `json:"pos"`
}

type Tokenizer struct {
	t *tokenizer.Tokenizer
}

func isNumericToken(text string) bool {
	if text == "" {
		return false
	}

	for _, r := range text {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

func NewTokenizer() (*Tokenizer, error) {
	t, err := tokenizer.New(ipa.Dict())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tokenizer: %w", err)
	}
	return &Tokenizer{t: t}, nil
}

func (tok *Tokenizer) Tokenize(text string) ([]Token, error) {
	if text == "" {
		return []Token{}, nil
	}

	tokens := tok.t.Tokenize(text)
	result := make([]Token, 0, len(tokens))

	for _, t := range tokens {
		pos := ""
		if len(t.POS()) > 0 {
			pos = t.POS()[0]
		}

		lemma := t.Surface
		if len(t.POS()) > 6 {
			lemma = t.POS()[6]
		}

		// skip punctuation and special BOS/EOS markers
		if t.Surface == "BOS" || t.Surface == "EOS" {
			continue
		}

		r, _ := utf8.DecodeRuneInString(t.Surface)
		if unicode.IsPunct(r) {
			continue
		}

		if isNumericToken(t.Surface) {
			continue
		}

		if t.Surface != "" {
			result = append(result, Token{
				Surface:      t.Surface,
				Lemma:        lemma,
				PartOfSpeech: pos,
			})
		}
	}

	return result, nil
}
