package analysis

import (
	"context"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestParseDictionaryQuery(t *testing.T) {
	query, err := parseDictionaryQuery(url.Values{
		"query":     {" 食 "},
		"jlpt":      {"5"},
		"page":      {"2"},
		"page_size": {"15"},
	})
	if err != nil {
		t.Fatalf("parseDictionaryQuery returned error: %v", err)
	}

	if query.Search != "食" || query.JLPTLevel == nil || *query.JLPTLevel != 5 {
		t.Fatalf("unexpected filters: %#v", query)
	}
	if !query.Paginated || query.Page != 2 || query.PageSize != 15 {
		t.Fatalf("unexpected pagination: %#v", query)
	}
}

func TestParseDictionaryQueryRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		values url.Values
	}{
		{name: "page below one", values: url.Values{"page": {"0"}}},
		{name: "page is not a number", values: url.Values{"page": {"x"}}},
		{name: "page size below one", values: url.Values{"page_size": {"0"}}},
		{name: "page size above maximum", values: url.Values{"page_size": {"101"}}},
		{name: "jlpt below one", values: url.Values{"jlpt": {"0"}}},
		{name: "jlpt above five", values: url.Values{"jlpt": {"6"}}},
		{name: "query too long", values: url.Values{"query": {string(make([]rune, 101))}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseDictionaryQuery(test.values); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSearchDictionaryFiltersAndPaginates(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	level := 5
	mock.ExpectQuery(`SELECT count\(\*\) FROM "japanese_dictionary"`).
		WithArgs(`%\%\_%`, `%\%\_%`, `%\%\_%`, level).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(16))
	mock.ExpectQuery(`SELECT kanji, hiragana, meaning, jlpt_level FROM "japanese_dictionary"`).
		WithArgs(`%\%\_%`, `%\%\_%`, `%\%\_%`, level, 15, 15).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "meaning", "jlpt_level"}).
			AddRow("食べる", "たべる", "to eat", 5))

	page, err := repo.SearchDictionary(context.Background(), "ja", DictionaryQuery{
		Search:    "%_",
		JLPTLevel: &level,
		Page:      2,
		PageSize:  15,
		Paginated: true,
	})
	if err != nil {
		t.Fatalf("SearchDictionary returned error: %v", err)
	}

	if page.Total != 16 || page.TotalPages != 2 || page.Page != 2 || page.PageSize != 15 {
		t.Fatalf("unexpected pagination: %#v", page)
	}
	if len(page.Entries) != 1 || page.Entries[0].Kanji != "食べる" {
		t.Fatalf("unexpected entries: %#v", page.Entries)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSearchDictionaryClampsPageAfterResultsShrink(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(16))
	mock.ExpectQuery(`SELECT kanji, hiragana, meaning, jlpt_level FROM "japanese_dictionary"`).
		WithArgs(15, 15).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "meaning", "jlpt_level"}))

	page, err := repo.SearchDictionary(context.Background(), "ja", DictionaryQuery{
		Page:      3,
		PageSize:  15,
		Paginated: true,
	})
	if err != nil {
		t.Fatalf("SearchDictionary returned error: %v", err)
	}
	if page.Page != 2 || page.TotalPages != 2 {
		t.Fatalf("page was not clamped: %#v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSearchDictionaryClampsEmptyResultsToFirstPage(t *testing.T) {
	repo, mock, cleanup := newSQLMockRepository(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "japanese_dictionary"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT kanji, hiragana, meaning, jlpt_level FROM "japanese_dictionary"`).
		WithArgs(15).
		WillReturnRows(sqlmock.NewRows([]string{"kanji", "hiragana", "meaning", "jlpt_level"}))

	page, err := repo.SearchDictionary(context.Background(), "ja", DictionaryQuery{
		Page:      3,
		PageSize:  15,
		Paginated: true,
	})
	if err != nil {
		t.Fatalf("SearchDictionary returned error: %v", err)
	}
	if page.Page != 1 || page.TotalPages != 1 || page.Total != 0 {
		t.Fatalf("empty result page was not clamped: %#v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
