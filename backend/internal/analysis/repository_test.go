package analysis

import (
	"context"
	"errors"
	"testing"

	"obucon/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

func TestGetKnownLemmasReturnsEmptyWithoutQuerying(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	known, err := repo.GetKnownLemmas(context.Background(), 1, "ja", nil)
	if err != nil {
		t.Fatalf("GetKnownLemmas returned error: %v", err)
	}
	if len(known) != 0 {
		t.Fatalf("expected empty lookup, got %#v", known)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetKnownLemmasOnlyMarksReturnedRows(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT "lemma" FROM "known_words"`).
		WithArgs(7, "ja", "食べる", "飲む", "走る").
		WillReturnRows(sqlmock.NewRows([]string{"lemma"}).AddRow("食べる").AddRow("走る"))

	known, err := repo.GetKnownLemmas(context.Background(), 7, "ja", []string{"食べる", "飲む", "走る"})
	if err != nil {
		t.Fatalf("GetKnownLemmas returned error: %v", err)
	}
	if !known["食べる"] || !known["走る"] {
		t.Fatalf("expected returned lemmas to be known, got %#v", known)
	}
	if known["飲む"] {
		t.Fatalf("expected 飲む to be unknown, got %#v", known)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetKnownLemmasWithDictionaryVariantsSkipsDictionaryWhenAllKnown(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT "lemma" FROM "known_words"`).
		WillReturnRows(sqlmock.NewRows([]string{"lemma"}).AddRow("食べる").AddRow("飲む"))

	known, err := repo.GetKnownLemmasWithDictionaryVariants(context.Background(), 7, "ja", []string{"食べる", "飲む"})
	if err != nil {
		t.Fatalf("GetKnownLemmasWithDictionaryVariants returned error: %v", err)
	}
	if len(known) != 2 {
		t.Fatalf("expected both lemmas known, got %#v", known)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetKnownLemmasWithDictionaryVariantsMatchesAcrossScriptForms(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT "lemma" FROM "known_words"`).
		WithArgs(7, "ja", "食べる").
		WillReturnRows(sqlmock.NewRows([]string{"lemma"}))
	mock.ExpectQuery(`SELECT kanji, hiragana FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana"}).AddRow("食べる", "たべる"))
	mock.ExpectQuery(`SELECT "lemma" FROM "known_words"`).
		WillReturnRows(sqlmock.NewRows([]string{"lemma"}).AddRow("たべる"))

	known, err := repo.GetKnownLemmasWithDictionaryVariants(context.Background(), 7, "ja", []string{"食べる"})
	if err != nil {
		t.Fatalf("GetKnownLemmasWithDictionaryVariants returned error: %v", err)
	}
	if !known["食べる"] {
		t.Fatalf("expected kanji form to be known via its hiragana reading, got %#v", known)
	}
	if !known["たべる"] {
		t.Fatalf("expected the matched variant itself to be known, got %#v", known)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetKnownLemmasWithDictionaryVariantsIgnoresRowsMissingAForm(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT "lemma" FROM "known_words"`).
		WillReturnRows(sqlmock.NewRows([]string{"lemma"}))
	mock.ExpectQuery(`SELECT kanji, hiragana FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana"}).AddRow("食べる", ""))

	known, err := repo.GetKnownLemmasWithDictionaryVariants(context.Background(), 7, "ja", []string{"食べる"})
	if err != nil {
		t.Fatalf("GetKnownLemmasWithDictionaryVariants returned error: %v", err)
	}
	if len(known) != 0 {
		t.Fatalf("expected no known lemmas, got %#v", known)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertKnownWordFillsGradeLevelFromDictionary(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT "jlpt_level" FROM "japanese_dictionary"`).
		WithArgs("食べる", "食べる", 1).
		WillReturnRows(sqlmock.NewRows([]string{"jlpt_level"}).AddRow(5))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "known_words"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	word := &models.KnownWord{UserID: 7, Language: "ja", Lemma: "食べる", Status: "known"}
	if err := repo.UpsertKnownWord(context.Background(), word); err != nil {
		t.Fatalf("UpsertKnownWord returned error: %v", err)
	}
	if word.GradeLevel == nil || *word.GradeLevel != 5 {
		t.Fatalf("expected grade level 5, got %#v", word.GradeLevel)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertKnownWordLeavesGradeLevelUnsetWhenDictionaryHasNoLevel(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT "jlpt_level" FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"jlpt_level"}).AddRow(nil))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "known_words"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	word := &models.KnownWord{UserID: 7, Language: "ja", Lemma: "ぬるぽ", Status: "known"}
	if err := repo.UpsertKnownWord(context.Background(), word); err != nil {
		t.Fatalf("UpsertKnownWord returned error: %v", err)
	}
	if word.GradeLevel != nil {
		t.Fatalf("expected grade level to stay nil, got %d", *word.GradeLevel)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertKnownWordSkipsDictionaryLookup(t *testing.T) {
	level := 3

	tests := []struct {
		name string
		word *models.KnownWord
	}{
		{
			name: "grade level already set",
			word: &models.KnownWord{UserID: 7, Language: "ja", Lemma: "食べる", GradeLevel: &level},
		},
		{
			name: "conjugation",
			word: &models.KnownWord{UserID: 7, Language: "ja", Lemma: "食べた", Metadata: []byte(`{"kind":"conjugation"}`)},
		},
		{
			name: "unsupported language",
			word: &models.KnownWord{UserID: 7, Language: "ko", Lemma: "먹다"},
		},
		{
			name: "empty lemma",
			word: &models.KnownWord{UserID: 7, Language: "ja"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock, cleanup := newSQLMockRepository(t)
			defer cleanup()

			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO "known_words"`).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			mock.ExpectCommit()

			if err := repo.UpsertKnownWord(context.Background(), tc.word); err != nil {
				t.Fatalf("UpsertKnownWord returned error: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

func TestGetDictionaryEntriesKeepsFirstRowPerLemma(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT kanji, hiragana, meaning, jlpt_level FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "meaning", "jlpt_level"}).
			AddRow("食べる", "たべる", "to eat", 5).
			AddRow("食べる", "たべる", "to consume", 4).
			AddRow("走る", "はしる", "to run", nil))

	entries, err := repo.GetDictionaryEntries(context.Background(), "ja", []string{"食べる", "走る"})
	if err != nil {
		t.Fatalf("GetDictionaryEntries returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %#v", entries)
	}
	if entries[0].Lemma != "食べる" || entries[0].Meaning != "to eat" {
		t.Fatalf("expected the first row to win, got %#v", entries[0])
	}
	if entries[1].JLPTLevel != nil {
		t.Fatalf("expected a nil JLPT level to survive, got %d", *entries[1].JLPTLevel)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetDictionaryEntriesSkipsRowsMatchingNoRequestedLemma(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT kanji, hiragana, meaning, jlpt_level FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "meaning", "jlpt_level"}).
			AddRow("飲む", "のむ", "to drink", 5))

	entries, err := repo.GetDictionaryEntries(context.Background(), "ja", []string{"食べる"})
	if err != nil {
		t.Fatalf("GetDictionaryEntries returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %#v", entries)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetDictionaryEntriesReturnsNilForUnsupportedLanguage(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	entries, err := repo.GetDictionaryEntries(context.Background(), "ko", []string{"먹다"})
	if err != nil {
		t.Fatalf("GetDictionaryEntries returned error: %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got %#v", entries)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetDictionaryMeaningsIndexesBothFormsAndKeepsTheFirst(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT kanji, hiragana, meaning FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "meaning"}).
			AddRow("食べる", "たべる", "to eat").
			AddRow("食べる", "たべる", "to consume"))

	meanings, err := repo.GetDictionaryMeanings(context.Background(), "ja", []string{"食べる"})
	if err != nil {
		t.Fatalf("GetDictionaryMeanings returned error: %v", err)
	}
	if meanings["食べる"] != "to eat" || meanings["たべる"] != "to eat" {
		t.Fatalf("expected both forms to map to the first meaning, got %#v", meanings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetDictionaryGradeLevelsKeepsTheHighestLevel(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT kanji, hiragana, jlpt_level FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "jlpt_level"}).
			AddRow("生", "せい", 3).
			AddRow("生", "なま", 5).
			AddRow("走る", "はしる", nil))

	levels, err := repo.GetDictionaryGradeLevels(context.Background(), "ja", []string{"生", "走る"})
	if err != nil {
		t.Fatalf("GetDictionaryGradeLevels returned error: %v", err)
	}
	if levels["生"] != 5 {
		t.Fatalf("expected the highest level to win, got %d", levels["生"])
	}
	if _, ok := levels["走る"]; ok {
		t.Fatalf("expected rows without a level to be skipped, got %#v", levels)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetDictionaryLookupsReturnEmptyWithoutQuerying(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	meanings, err := repo.GetDictionaryMeanings(context.Background(), "ja", nil)
	if err != nil {
		t.Fatalf("GetDictionaryMeanings returned error: %v", err)
	}
	if len(meanings) != 0 {
		t.Fatalf("expected no meanings, got %#v", meanings)
	}

	levels, err := repo.GetDictionaryGradeLevels(context.Background(), "ja", nil)
	if err != nil {
		t.Fatalf("GetDictionaryGradeLevels returned error: %v", err)
	}
	if len(levels) != 0 {
		t.Fatalf("expected no levels, got %#v", levels)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBulkAddKnownWordsByJLPTReportsInsertedRows(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO known_words`).
		WithArgs(7, "ja", 5, 5).
		WillReturnResult(sqlmock.NewResult(0, 42))

	inserted, err := repo.BulkAddKnownWordsByJLPT(context.Background(), 7, "ja", 5)
	if err != nil {
		t.Fatalf("BulkAddKnownWordsByJLPT returned error: %v", err)
	}
	if inserted != 42 {
		t.Fatalf("expected 42 inserted rows, got %d", inserted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateKnownWordReturnsNotFoundWhenNothingMatched(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE known_words`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.UpdateKnownWord(context.Background(), 7, "ja", "食べる", "to eat", 5)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpdateKnownWordSucceedsWhenARowMatched(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE known_words`).
		WithArgs(5, "to eat", 7, "ja", "食べる").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateKnownWord(context.Background(), 7, "ja", "食べる", "to eat", 5); err != nil {
		t.Fatalf("UpdateKnownWord returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRemoveKnownWordReturnsNotFoundWhenNothingMatched(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "known_words"`).
		WithArgs(7, "ja", "食べる").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err := repo.RemoveKnownWord(context.Background(), 7, "ja", "食べる")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListKnownWordsWithMeaningScansJapaneseRows(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`FROM known_words kw`).
		WithArgs(7, "ja").
		WillReturnRows(sqlmock.NewRows([]string{"lemma", "hiragana", "grade_level", "meaning", "kind"}).
			AddRow("食べる", "たべる", 5, "to eat", "").
			AddRow("食べた", "", nil, "past tense of 食べる", "conjugation"))

	entries, err := repo.ListKnownWordsWithMeaning(context.Background(), 7, "ja")
	if err != nil {
		t.Fatalf("ListKnownWordsWithMeaning returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %#v", entries)
	}
	if entries[0].Hiragana != "たべる" || entries[0].Meaning != "to eat" {
		t.Fatalf("unexpected first entry: %#v", entries[0])
	}
	if entries[0].GradeLevel == nil || *entries[0].GradeLevel != 5 {
		t.Fatalf("expected grade level 5, got %#v", entries[0].GradeLevel)
	}
	if entries[1].Kind != "conjugation" || entries[1].GradeLevel != nil {
		t.Fatalf("expected an ungraded conjugation, got %#v", entries[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestUpsertKnownWordPropagatesDictionaryLookupError(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	lookupErr := errors.New("dictionary unavailable")
	mock.ExpectQuery(`SELECT "jlpt_level" FROM "japanese_dictionary"`).
		WillReturnError(lookupErr)

	word := &models.KnownWord{UserID: 7, Language: "ja", Lemma: "食べる"}
	err := repo.UpsertKnownWord(context.Background(), word)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("expected the lookup error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestListKnownWordsWithMeaningFallsBackForUnsupportedLanguage(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT \* FROM "known_words"`).
		WithArgs(7, "ko").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "language", "lemma", "grade_level"}).
			AddRow(1, 7, "ko", "먹다", 2))

	entries, err := repo.ListKnownWordsWithMeaning(context.Background(), 7, "ko")
	if err != nil {
		t.Fatalf("ListKnownWordsWithMeaning returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Lemma != "먹다" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
	if entries[0].Meaning != "" || entries[0].Hiragana != "" {
		t.Fatalf("expected the fallback to leave meaning and hiragana empty, got %#v", entries[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
