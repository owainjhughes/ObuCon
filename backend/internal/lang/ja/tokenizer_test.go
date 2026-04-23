package ja

import (
	"sync"
	"testing"
)

func TestIsNumericToken(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"0", true},
		{"123", true},
		{"12a", false},
		{"a12", false},
		{"-1", false},
		{"１２３", true}, // full-width digits (Unicode Nd category)
	}
	for _, tc := range cases {
		if got := isNumericToken(tc.in); got != tc.want {
			t.Errorf("isNumericToken(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsKatakanaToken(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"コーヒー", true},
		{"カタカナ", true},
		{"ア", true},
		{"カこ", false},   // mixed katakana + hiragana → false
		{"これ", false},   // pure hiragana → false
		{"hello", false}, // roman → false
		{"123", false},   // numeric → false
		{"ー", true},      // prolonged vowel mark alone (within katakana range)
	}
	for _, tc := range cases {
		if got := isKatakanaToken(tc.in); got != tc.want {
			t.Errorf("isKatakanaToken(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsRomanToken(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"hello", true},
		{"Hello", true},
		{"don't", true},
		{"abc123", true},
		{"123", false}, // digits only, no letter
		{"コーヒー", false},
		{"hello!", false}, // exclamation not allowed
		{"a b", false},    // space not allowed
	}
	for _, tc := range cases {
		if got := isRomanToken(tc.in); got != tc.want {
			t.Errorf("isRomanToken(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// Kagome's IPA dict load is slow (~10 MB); share one Tokenizer across subtests.
var (
	sharedTokOnce sync.Once
	sharedTok     *Tokenizer
	sharedTokErr  error
)

func getSharedTokenizer(t *testing.T) *Tokenizer {
	t.Helper()
	sharedTokOnce.Do(func() {
		sharedTok, sharedTokErr = NewTokenizer()
	})
	if sharedTokErr != nil {
		t.Fatalf("NewTokenizer: %v", sharedTokErr)
	}
	return sharedTok
}

func TestNewTokenizer(t *testing.T) {
	tok, err := NewTokenizer()
	if err != nil {
		t.Fatalf("NewTokenizer returned error: %v", err)
	}
	if tok == nil {
		t.Fatal("NewTokenizer returned nil tokenizer")
	}
}

func TestTokenize_Empty(t *testing.T) {
	tok := getSharedTokenizer(t)
	got, err := tok.Tokenize("")
	if err != nil {
		t.Fatalf("Tokenize(\"\"): unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Tokenize(\"\"): got %d tokens, want 0", len(got))
	}
}

func TestTokenize_FiltersPunctuationAndBosEos(t *testing.T) {
	tok := getSharedTokenizer(t)
	tokens, err := tok.Tokenize("私はコーヒーを飲みます。")
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	if len(tokens) == 0 {
		t.Fatal("expected at least one token")
	}
	for _, tk := range tokens {
		if tk.Surface == "BOS" || tk.Surface == "EOS" {
			t.Errorf("BOS/EOS marker leaked into output: %+v", tk)
		}
		if tk.Surface == "。" || tk.Surface == "、" {
			t.Errorf("punctuation leaked into output: %+v", tk)
		}
		if tk.Lemma == "" {
			t.Errorf("token has empty lemma: %+v", tk)
		}
	}
}

func TestTokenize_Deterministic(t *testing.T) {
	tok := getSharedTokenizer(t)
	input := "私はコーヒーを飲みます。"
	first, err := tok.Tokenize(input)
	if err != nil {
		t.Fatalf("first Tokenize: %v", err)
	}
	second, err := tok.Tokenize(input)
	if err != nil {
		t.Fatalf("second Tokenize: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("token count differs: first=%d second=%d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("token %d differs: first=%+v second=%+v", i, first[i], second[i])
		}
	}
}

func TestTokenize_KatakanaFlagSet(t *testing.T) {
	tok := getSharedTokenizer(t)
	tokens, err := tok.Tokenize("コーヒー")
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	// "コーヒー" should appear as a single katakana-flagged token.
	found := false
	for _, tk := range tokens {
		if tk.Surface == "コーヒー" && tk.IsKatakana {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a katakana-flagged token 'コーヒー' in %+v", tokens)
	}
}
