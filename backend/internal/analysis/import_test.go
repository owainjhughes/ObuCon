package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"obucon/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSQLMockRepository(t *testing.T) (*Repository, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("gorm.Open: %v", err)
	}

	return NewRepository(db), mock, func() { sqlDB.Close() }
}

func TestNormalizeVocabularyImportEntriesTrimsAndUsesLastDuplicate(t *testing.T) {
	level := 5
	entries, err := normalizeVocabularyImportEntries([]VocabularyImportEntry{
		{Lemma: " 食べる ", Meaning: " old "},
		{Lemma: "食べる", Hiragana: " たべる ", Meaning: " to eat ", JLPTLevel: &level},
	})
	if err != nil {
		t.Fatalf("normalizeVocabularyImportEntries returned error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	got := entries[0]
	if got.Lemma != "食べる" || got.Hiragana != "たべる" || got.Meaning != "to eat" {
		t.Fatalf("unexpected normalized entry: %#v", got)
	}
	if got.JLPTLevel == nil || *got.JLPTLevel != 5 {
		t.Fatalf("unexpected JLPT level: %#v", got.JLPTLevel)
	}
}

func TestNormalizeVocabularyImportEntriesRejectsBlankLemma(t *testing.T) {
	if _, err := normalizeVocabularyImportEntries([]VocabularyImportEntry{{Lemma: "  "}}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGetDictionaryGradeLevelsKeepsEasiestMatchingLevel(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT kanji, hiragana, jlpt_level FROM "japanese_dictionary"`).
		WithArgs("言葉", "言葉").
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "jlpt_level"}).
			AddRow("言葉", "げんよう", 5).
			AddRow("言葉", "ことば", 3))

	levels, err := repo.GetDictionaryGradeLevels(context.Background(), "ja", []string{"言葉"})
	if err != nil {
		t.Fatalf("GetDictionaryGradeLevels returned error: %v", err)
	}
	if levels["言葉"] != 5 {
		t.Fatalf("got JLPT N%d, want N5", levels["言葉"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestImportedKnownWordPreservesFlexibleMetadataThroughPatchOnly(t *testing.T) {
	entry := VocabularyImportEntry{Lemma: "言葉", Meaning: "word"}
	existing := models.KnownWord{
		Status:   "known",
		Metadata: []byte(`{"frequency":2,"source":{"name":"custom"},"meaning":"old"}`),
	}

	word, changed, err := importedKnownWord(7, "ja", entry, &existing, 0)
	if err != nil {
		t.Fatalf("importedKnownWord returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed metadata")
	}

	var patch map[string]any
	if err := json.Unmarshal(word.Metadata, &patch); err != nil {
		t.Fatalf("decode metadata patch: %v", err)
	}
	if len(patch) != 1 || patch["meaning"] != "word" {
		t.Fatalf("unexpected metadata patch: %#v", patch)
	}

	existing.Metadata = []byte(`null`)
	if _, _, err := importedKnownWord(7, "ja", entry, &existing, 0); err != nil {
		t.Fatalf("JSON null metadata returned error: %v", err)
	}
}

func TestImportVocabularyCommitsAddedUpdatedAndSkippedEntries(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	level := 5
	entries := []VocabularyImportEntry{
		{Lemma: "新しい", Meaning: "new", JLPTLevel: &level},
		{Lemma: "既知", Meaning: "updated", JLPTLevel: &level},
		{Lemma: "同じ", Meaning: "same", JLPTLevel: &level},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "known_words"`).
		WithArgs(uint(7), "ja", "新しい", "既知", "同じ").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "language", "lemma", "grade_level", "status", "metadata"}).
			AddRow(11, 7, "ja", "既知", 5, "known", []byte(`{"meaning":"old"}`)).
			AddRow(12, 7, "ja", "同じ", 5, "known", []byte(`{"meaning":"same"}`)))
	mock.ExpectQuery(`INSERT INTO "known_words".*COALESCE\(known_words.metadata, '\{\}'::jsonb\) \|\| COALESCE\(EXCLUDED.metadata, '\{\}'::jsonb\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(13).AddRow(11))
	mock.ExpectCommit()

	result, err := repo.ImportVocabulary(context.Background(), 7, "ja", entries)
	if err != nil {
		t.Fatalf("ImportVocabulary returned error: %v", err)
	}
	if *result != (VocabularyImportResult{Added: 1, Updated: 1, Skipped: 1}) {
		t.Fatalf("unexpected result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestImportVocabularyRollsBackWhenWriteFails(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	entries := []VocabularyImportEntry{{Lemma: "新しい", Meaning: "new"}}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT \* FROM "known_words"`).
		WithArgs(uint(7), "ja", "新しい").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "language", "lemma", "grade_level", "status", "metadata"}))
	mock.ExpectQuery(`SELECT kanji, hiragana, jlpt_level FROM "japanese_dictionary"`).
		WithArgs("新しい", "新しい").
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "jlpt_level"}))
	mock.ExpectQuery(`INSERT INTO "known_words"`).WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	if _, err := repo.ImportVocabulary(context.Background(), 7, "ja", entries); err == nil {
		t.Fatal("expected write failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestImportVocabularyHandlerSeparatesValidationAndPersistenceErrors(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()
	handler := NewAnalysisHandler(NewService(nil, repo))

	t.Run("validation error", func(t *testing.T) {
		response := performVocabularyImportRequest(t, handler, `{"language":"ja","entries":[{"lemma":" "}]}`)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want 400", response.Code)
		}
		if !strings.Contains(response.Body.String(), "lemma cannot be empty") {
			t.Fatalf("unexpected response: %s", response.Body.String())
		}
	})

	t.Run("rejected by binding", func(t *testing.T) {
		bodies := map[string]string{
			"no entries":   `{"language":"ja","entries":[]}`,
			"blank lemma":  `{"language":"ja","entries":[{"lemma":""}]}`,
			"bad jlpt":     `{"language":"ja","entries":[{"lemma":"word","jlpt_level":6}]}`,
			"bad language": `{"language":"jpn","entries":[{"lemma":"word"}]}`,
		}
		for name, body := range bodies {
			t.Run(name, func(t *testing.T) {
				if response := performVocabularyImportRequest(t, handler, body); response.Code != http.StatusBadRequest {
					t.Fatalf("got status %d, want 400", response.Code)
				}
			})
		}
	})

	t.Run("oversized request", func(t *testing.T) {
		body := `{"language":"ja","entries":[{"lemma":"word","meaning":"` + strings.Repeat("x", maxVocabularyImportBodyBytes) + `"}]}`
		response := performVocabularyImportRequest(t, handler, body)
		if response.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("got status %d, want 413", response.Code)
		}
	})

	t.Run("persistence error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT \* FROM "known_words"`).
			WithArgs(uint(7), "ja", "word").
			WillReturnError(errors.New("driver connection secret"))
		mock.ExpectRollback()

		response := performVocabularyImportRequest(t, handler, `{"language":"ja","entries":[{"lemma":"word"}]}`)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want 500", response.Code)
		}
		if response.Body.String() != `{"error":"failed to import vocabulary"}` {
			t.Fatalf("unexpected response: %s", response.Body.String())
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func performVocabularyImportRequest(t *testing.T, handler *AnalysisHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Set("userID", uint(7))
	context.Request = httptest.NewRequest(http.MethodPost, "/vocab/import", bytes.NewBufferString(body))
	context.Request.Header.Set("Content-Type", "application/json")
	handler.ImportVocabulary(context)
	return response
}
