package helpers

import (
	"bytes"
	"mime/multipart"
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

func TestExtractTextFromFileHeader_NonUTF8Rejected(t *testing.T) {
	// Invalid UTF-8 bytes (lone continuation byte)
	fh := buildFileHeader(t, "bad.txt", []byte{0xff, 0xfe, 0x00})
	_, err := ExtractTextFromFileHeader(fh)
	if err == nil {
		t.Fatal("expected error for non-UTF-8 content, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("expected 'UTF-8' in error, got: %v", err)
	}
}
