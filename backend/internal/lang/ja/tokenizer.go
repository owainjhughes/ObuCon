package ja

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

type Token struct {
	Surface       string `json:"surface"`
	Lemma         string `json:"lemma"`
	PartOfSpeech  string `json:"pos"`
	IsKatakana    bool   `json:"is_katakana"`
	IsRoman       bool   `json:"is_roman"`
	IsNonJapanese bool   `json:"is_non_japanese"`
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

func isKatakanaToken(text string) bool {
	if text == "" {
		return false
	}
	// Returns true only if the token contains katakana (U+30A0–U+30FF) and no hiragana.
	hasKatakana := false
	for _, r := range text {
		if (r >= 0x30A0 && r <= 0x30FF) || r == 0x30FC { // katakana range or prolonged vowel mark
			hasKatakana = true
		} else if (r >= 0x3040 && r <= 0x309F) || r == 0x3099 || r == 0x309A {
			// hiragana present, not a pure katakana token
			return false
		}
	}
	return hasKatakana
}

func containsJapaneseChar(text string) bool {
	for _, r := range text {
		switch {
		case r == 0x3005, // 々 kanji iteration mark
			r >= 0x3040 && r <= 0x309F, // hiragana
			r >= 0x30A0 && r <= 0x30FF, // katakana (incl. ・ and ー)
			r >= 0x3400 && r <= 0x4DBF, // CJK Extension A
			r >= 0x4E00 && r <= 0x9FFF, // CJK Unified Ideographs
			r >= 0xFF65 && r <= 0xFF9F: // halfwidth katakana
			return true
		}
	}
	return false
}

func isRomanToken(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '\'' || (r >= '0' && r <= '9')) {
			return false
		}
	}
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
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
		pos := strings.Join(t.POS(), " ")

		lemma := t.Surface
		if base, ok := t.BaseForm(); ok && base != "" {
			lemma = base
		} else if len(t.POS()) > 6 {
			lemma = t.POS()[6]
		}

		// skip punctuation, whitespace, and special BOS/EOS markers
		if t.Surface == "BOS" || t.Surface == "EOS" {
			continue
		}

		if strings.TrimSpace(t.Surface) == "" {
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
				Surface:       t.Surface,
				Lemma:         lemma,
				PartOfSpeech:  pos,
				IsKatakana:    isKatakanaToken(t.Surface),
				IsRoman:       isRomanToken(t.Surface),
				IsNonJapanese: !containsJapaneseChar(t.Surface) || strings.Contains(pos, "記号"),
			})
		}
	}

	return result, nil
}
