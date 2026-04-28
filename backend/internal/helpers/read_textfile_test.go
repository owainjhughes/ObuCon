package helpers

import (
	"bytes"
	"mime/multipart"
	"obucon/internal/lang/ja"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeTextForAnalysis(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "collapses CRLF and CR to LF",
			in:   "a\r\nb\rc",
			want: "a\nb\nc",
		},
		{
			name: "trims trailing whitespace per line",
			in:   "hello   \nworld\t\t",
			want: "hello\nworld",
		},
		{
			name: "caps consecutive blanks at two",
			in:   "a\n\n\n\n\nb",
			want: "a\n\n\nb",
		},
		{
			name: "trims leading and trailing whitespace of result",
			in:   "\n\n  hello  \n\n",
			want: "hello",
		},
		{
			name: "simple single line unchanged",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTextForAnalysis(tc.in)
			if got != tc.want {
				t.Errorf("normalizeTextForAnalysis(%q)\n got  %q\n want %q", tc.in, got, tc.want)
			}
		})
	}
}

// buildFileHeader constructs an in-memory *multipart.FileHeader for a file
// named `filename` with the given content. Size is set from the part bytes.
func buildFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content)) + 4096)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	headers := form.File["file"]
	if len(headers) != 1 {
		t.Fatalf("expected 1 file header, got %d", len(headers))
	}
	return headers[0]
}

func TestExtractTextFromFileHeader_UnsupportedExtension(t *testing.T) {
	fh := buildFileHeader(t, "malware.exe", []byte("whatever"))
	_, err := ExtractTextFromFileHeader(fh)
	if err == nil {
		t.Fatal("expected error for .exe, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("expected 'unsupported file type' in error, got: %v", err)
	}
}

func TestExtractTextFromFileHeader_NilHeader(t *testing.T) {
	_, err := ExtractTextFromFileHeader(nil)
	if err == nil {
		t.Fatal("expected error for nil header, got nil")
	}
}

func TestExtractTextFromFileHeader_Empty(t *testing.T) {
	fh := buildFileHeader(t, "empty.txt", []byte{})
	_, err := ExtractTextFromFileHeader(fh)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' in error, got: %v", err)
	}
}

func TestExtractTextFromFileHeader_PlainText(t *testing.T) {
	content := []byte("hello world\nsecond line\n")
	fh := buildFileHeader(t, "note.txt", content)
	result, err := ExtractTextFromFileHeader(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Filename != "note.txt" {
		t.Errorf("Filename: got %q, want %q", result.Filename, "note.txt")
	}
	// Normalization strips trailing newline.
	want := "hello world\nsecond line"
	if result.Text != want {
		t.Errorf("Text: got %q, want %q", result.Text, want)
	}
	if result.ExtractedChars != len([]rune(want)) {
		t.Errorf("ExtractedChars: got %d, want %d", result.ExtractedChars, len([]rune(want)))
	}
}

func TestExtractTextFromFileHeader_MarkdownAccepted(t *testing.T) {
	fh := buildFileHeader(t, "readme.md", []byte("# title\n\nbody"))
	result, err := ExtractTextFromFileHeader(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Text, "title") {
		t.Errorf("expected 'title' in extracted text, got %q", result.Text)
	}
}

func TestExtractTextFromFileHeader_PdfTokenParity(t *testing.T) {
	testCasesDir := filepath.Join("..", "..", "..", "..", "project", "research", "test-cases")
	if _, err := os.Stat(testCasesDir); os.IsNotExist(err) {
		t.Skipf("test-cases directory not found at %s; skipping", testCasesDir)
	}

	tokenizer, err := ja.NewTokenizer()
	if err != nil {
		t.Fatalf("NewTokenizer: %v", err)
	}

	formats := []string{"tenyomi1.txt", "tenyomi1.md", "tenyomi1.docx", "tenyomi1.pdf"}
	counts := make(map[string]int, len(formats))
	tokenSets := make(map[string]map[string]struct{}, len(formats))

	for _, name := range formats {
		path := filepath.Join(testCasesDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fh := buildFileHeader(t, name, raw)
		extracted, err := ExtractTextFromFileHeader(fh)
		if err != nil {
			t.Fatalf("ExtractTextFromFileHeader(%s): %v", name, err)
		}
		tokens, err := tokenizer.Tokenize(extracted.Text)
		if err != nil {
			t.Fatalf("Tokenize(%s): %v", name, err)
		}
		counts[name] = len(tokens)
		set := make(map[string]struct{}, len(tokens))
		for _, tok := range tokens {
			set[tok.Surface] = struct{}{}
		}
		tokenSets[name] = set
		newlineCount := strings.Count(extracted.Text, "\n")
		t.Logf("%s: %d tokens, %d newlines, %d chars", name, len(tokens), newlineCount, extracted.ExtractedChars)
	}

	txt := counts["tenyomi1.txt"]
	pdf := counts["tenyomi1.pdf"]
	if txt == 0 {
		t.Fatal("txt token count is zero; test data missing")
	}

	const tolerance = 5
	diff := pdf - txt
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		var only []string
		for surface := range tokenSets["tenyomi1.pdf"] {
			if _, ok := tokenSets["tenyomi1.txt"][surface]; !ok {
				only = append(only, surface)
			}
		}
		t.Errorf("pdf vs txt token count diff %d exceeds tolerance %d (pdf=%d txt=%d). Tokens unique to pdf: %v",
			diff, tolerance, pdf, txt, only)
	}
}

func TestStripCJKInternalNewlines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "removes newline between two kanji",
			in:   "知\nる",
			want: "知る",
		},
		{
			name: "removes newline between hiragana and kanji",
			in:   "知ら\nれている",
			want: "知られている",
		},
		{
			name: "removes newline between katakana characters",
			in:   "コー\nヒー",
			want: "コーヒー",
		},
		{
			name: "preserves newline between ASCII letters",
			in:   "hello\nworld",
			want: "hello\nworld",
		},
		{
			name: "preserves newline between CJK and ASCII",
			in:   "日本\nABC",
			want: "日本\nABC",
		},
		{
			name: "preserves paragraph break (double newline) between CJK",
			in:   "日本\n\n語",
			want: "日本\n\n語",
		},
		{
			name: "preserves leading and trailing newlines",
			in:   "\n日本\n",
			want: "\n日本\n",
		},
		{
			name: "no newlines passthrough",
			in:   "日本語",
			want: "日本語",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "mixed: drops CJK-internal but keeps ASCII-internal",
			in:   "日本\n語 hello\nworld",
			want: "日本語 hello\nworld",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripCJKInternalNewlines(tc.in)
			if got != tc.want {
				t.Errorf("stripCJKInternalNewlines(%q)\n got  %q\n want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtractTextFromFileHeader_NonUTF8Rejected(t *testing.T) {
	fh := buildFileHeader(t, "bad.txt", []byte{0xff, 0xfe, 0x00})
	_, err := ExtractTextFromFileHeader(fh)
	if err == nil {
		t.Fatal("expected error for non-UTF-8 content, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("expected 'UTF-8' in error, got: %v", err)
	}
}
