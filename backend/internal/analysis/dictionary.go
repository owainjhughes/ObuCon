package analysis

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"obucon/internal/models"

	"gorm.io/gorm"
)

const (
	defaultDictionaryPageSize = 15
	maxDictionaryPageSize     = 100
	maxDictionaryPage         = 100000
	maxDictionaryQueryRunes   = 100
)

type DictionaryQuery struct {
	Search    string
	JLPTLevel *int
	Page      int
	PageSize  int
}

type DictionaryPage struct {
	Entries    []DictionaryEntry `json:"entries"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	Total      int64             `json:"total"`
	TotalPages int               `json:"total_pages"`
}

func parseDictionaryQuery(values url.Values) (DictionaryQuery, error) {
	query := DictionaryQuery{
		Search:   strings.TrimSpace(values.Get("query")),
		Page:     1,
		PageSize: defaultDictionaryPageSize,
	}

	if utf8.RuneCountInString(query.Search) > maxDictionaryQueryRunes {
		return DictionaryQuery{}, fmt.Errorf("query must be at most %d characters", maxDictionaryQueryRunes)
	}

	if raw := values.Get("page"); raw != "" {
		page, err := parsePositiveQueryInt(raw, "page", 1, maxDictionaryPage)
		if err != nil {
			return DictionaryQuery{}, err
		}
		query.Page = page
	}

	if raw := values.Get("page_size"); raw != "" {
		pageSize, err := parsePositiveQueryInt(raw, "page_size", 1, maxDictionaryPageSize)
		if err != nil {
			return DictionaryQuery{}, err
		}
		query.PageSize = pageSize
	}

	if raw := strings.TrimSpace(values.Get("jlpt")); raw != "" {
		level, err := parsePositiveQueryInt(raw, "jlpt", 1, 5)
		if err != nil {
			return DictionaryQuery{}, err
		}
		query.JLPTLevel = &level
	}

	return query, nil
}

func parsePositiveQueryInt(raw, name string, minimum, maximum int) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("%s must be between %d and %d", name, minimum, maximum)
	}
	return value, nil
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func (s *Service) SearchDictionary(ctx context.Context, language string, query DictionaryQuery) (*DictionaryPage, error) {
	return s.repo.SearchDictionary(ctx, language, query)
}

func (r *Repository) SearchDictionary(ctx context.Context, language string, query DictionaryQuery) (*DictionaryPage, error) {
	page := &DictionaryPage{
		Entries:    []DictionaryEntry{},
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: 1,
	}
	if language != "ja" {
		return page, nil
	}

	applyFilters := func(db *gorm.DB) *gorm.DB {
		if query.Search != "" {
			pattern := "%" + escapeLikePattern(query.Search) + "%"
			db = db.Where("(kanji ILIKE ? OR hiragana ILIKE ? OR meaning ILIKE ?)", pattern, pattern, pattern)
		}
		if query.JLPTLevel != nil {
			db = db.Where("jlpt_level = ?", *query.JLPTLevel)
		}
		return db
	}

	if err := applyFilters(r.db.WithContext(ctx).Model(&models.JapaneseDictionary{})).Count(&page.Total).Error; err != nil {
		return nil, err
	}
	if page.Total > 0 {
		page.TotalPages = int((page.Total + int64(query.PageSize) - 1) / int64(query.PageSize))
	}

	err := applyFilters(r.db.WithContext(ctx).
		Model(&models.JapaneseDictionary{}).
		Select("kanji, hiragana, meaning, jlpt_level")).
		Order("jlpt_level DESC NULLS LAST, kanji, hiragana, id").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&page.Entries).Error
	if err != nil {
		return nil, err
	}

	return page, nil
}
