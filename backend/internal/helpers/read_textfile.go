package helpers

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

const (
	MaxUploadFileBytes    int64 = 5 * 1024 * 1024
	MaxExtractedTextChars       = 200000
)

type TextFileExtraction struct {
	Filename       string
	Bytes          int64
	Text           string
	ExtractedChars int
}

func ExtractTextFromFileHeader(fileHeader *multipart.FileHeader) (*TextFileExtraction, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file is required")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".txt" && ext != ".md" && ext != ".docx" && ext != ".pdf" {
		return nil, fmt.Errorf("unsupported file type: only .txt, .md, .docx, and .pdf are allowed")
	}

	if fileHeader.Size <= 0 {
		return nil, fmt.Errorf("file is empty")
	}

	if fileHeader.Size > MaxUploadFileBytes {
		return nil, fmt.Errorf("file is too large: maximum size is 5MB")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to read file")
	}
	defer file.Close()

	raw, err := io.ReadAll(io.LimitReader(file, MaxUploadFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file")
	}

	if int64(len(raw)) > MaxUploadFileBytes {
		return nil, fmt.Errorf("file is too large: maximum size is 5MB")
	}

	var extractedText string
	switch ext {
	case ".txt", ".md":
		if !utf8.Valid(raw) {
			return nil, fmt.Errorf("file must be UTF-8 encoded text")
		}
		extractedText = string(raw)
	case ".docx":
		extractedText, err = extractDocxText(raw)
		if err != nil {
			return nil, err
		}
	case ".pdf":
		extractedText, err = extractPdfText(raw)
		if err != nil {
			return nil, err
		}
	}

	normalizedText := NormalizeTextForAnalysis(extractedText)
	if strings.TrimSpace(normalizedText) == "" {
		return nil, fmt.Errorf("file contains no readable text")
	}

	runeCount := utf8.RuneCountInString(normalizedText)
	if runeCount > MaxExtractedTextChars {
		return nil, fmt.Errorf("extracted text is too long")
	}

	return &TextFileExtraction{
		Filename:       fileHeader.Filename,
		Bytes:          int64(len(raw)),
		Text:           normalizedText,
		ExtractedChars: runeCount,
	}, nil
}

func extractDocxText(data []byte) (string, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to parse .docx file: not a valid Word document")
	}

	for _, f := range zipReader.File {
		if f.Name != "word/document.xml" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("failed to read .docx content")
		}
		defer rc.Close()

		decoder := xml.NewDecoder(rc)
		var text strings.Builder
		inText := false

		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("failed to parse .docx content")
			}

			switch t := token.(type) {
			case xml.StartElement:
				switch t.Name.Local {
				case "t":
					inText = true
				case "p":
					text.WriteByte('\n')
				case "br":
					text.WriteByte('\n')
				}
			case xml.EndElement:
				if t.Name.Local == "t" {
					inText = false
				}
			case xml.CharData:
				if inText {
					text.Write([]byte(t))
				}
			}
		}

		return text.String(), nil
	}

	return "", fmt.Errorf("failed to parse .docx file: no text content found")
}

func extractPdfText(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to parse .pdf file")
	}

	var text strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text.WriteString(pageText)
	}

	return text.String(), nil
}

func NormalizeTextForAnalysis(text string) string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	var builder strings.Builder
	blankStreak := 0

	for i, line := range lines {
		cleanLine := strings.TrimRight(line, " \t")
		if cleanLine == "" {
			blankStreak++
			if blankStreak > 2 {
				continue
			}
		} else {
			blankStreak = 0
		}

		if i > 0 && builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(cleanLine)
	}

	return strings.TrimSpace(builder.String())
}
