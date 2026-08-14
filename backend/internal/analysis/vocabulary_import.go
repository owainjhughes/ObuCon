package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"obucon/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxVocabularyImportEntries   = 500
	maxVocabularyImportBodyBytes = 4 << 20
)

type VocabularyImportEntry struct {
	Lemma     string `json:"lemma"`
	Hiragana  string `json:"hiragana,omitempty"`
	Meaning   string `json:"meaning,omitempty"`
	JLPTLevel *int   `json:"jlpt_level,omitempty"`
}

type VocabularyImportResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

type vocabularyImportValidationError struct {
	message string
}

func (e *vocabularyImportValidationError) Error() string {
	return e.message
}

func invalidVocabularyImport(format string, args ...any) error {
	return &vocabularyImportValidationError{message: fmt.Sprintf(format, args...)}
}

func normalizeVocabularyImportEntries(entries []VocabularyImportEntry) ([]VocabularyImportEntry, error) {
	if len(entries) == 0 {
		return nil, invalidVocabularyImport("at least one entry is required")
	}
	if len(entries) > maxVocabularyImportEntries {
		return nil, invalidVocabularyImport("at most %d entries can be imported", maxVocabularyImportEntries)
	}

	normalized := make([]VocabularyImportEntry, 0, len(entries))
	indexByLemma := make(map[string]int, len(entries))
	for i, entry := range entries {
		entry.Lemma = strings.TrimSpace(entry.Lemma)
		entry.Hiragana = strings.TrimSpace(entry.Hiragana)
		entry.Meaning = strings.TrimSpace(entry.Meaning)
		if entry.Lemma == "" {
			return nil, invalidVocabularyImport("entry %d: lemma cannot be empty", i+1)
		}
		if entry.JLPTLevel != nil && (*entry.JLPTLevel < 1 || *entry.JLPTLevel > 5) {
			return nil, invalidVocabularyImport("entry %d: jlpt level must be between 1 and 5", i+1)
		}

		if index, exists := indexByLemma[entry.Lemma]; exists {
			normalized[index] = entry
			continue
		}
		indexByLemma[entry.Lemma] = len(normalized)
		normalized = append(normalized, entry)
	}

	return normalized, nil
}

func (s *Service) ImportVocabulary(ctx context.Context, userID uint, language string, entries []VocabularyImportEntry) (*VocabularyImportResult, error) {
	if len(language) != 2 {
		return nil, invalidVocabularyImport("language must be a 2-character code")
	}
	normalized, err := normalizeVocabularyImportEntries(entries)
	if err != nil {
		return nil, err
	}
	return s.repo.ImportVocabulary(ctx, userID, language, normalized)
}

func (r *Repository) ImportVocabulary(ctx context.Context, userID uint, language string, entries []VocabularyImportEntry) (*VocabularyImportResult, error) {
	result := &VocabularyImportResult{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lemmas := make([]string, len(entries))
		for i, entry := range entries {
			lemmas[i] = entry.Lemma
		}

		var existingRows []models.KnownWord
		if err := tx.Where("user_id = ? AND language = ? AND lemma IN ?", userID, language, lemmas).Find(&existingRows).Error; err != nil {
			return err
		}
		existingByLemma := make(map[string]models.KnownWord, len(existingRows))
		for _, row := range existingRows {
			existingByLemma[row.Lemma] = row
		}

		gradeLevels, err := dictionaryImportGradeLevels(tx, language, entries, existingByLemma)
		if err != nil {
			return err
		}

		changed := make([]models.KnownWord, 0, len(entries))
		for _, entry := range entries {
			existing, found := existingByLemma[entry.Lemma]
			word, didChange, err := importedKnownWord(userID, language, entry, existing, found, gradeLevels[entry.Lemma])
			if err != nil {
				return err
			}
			if !found {
				result.Added++
			} else if didChange {
				result.Updated++
			} else {
				result.Skipped++
			}
			if !found || didChange {
				changed = append(changed, word)
			}
		}

		if len(changed) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "language"}, {Name: "lemma"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"grade_level": gorm.Expr("COALESCE(EXCLUDED.grade_level, known_words.grade_level)"),
				"metadata":    gorm.Expr("CASE WHEN jsonb_typeof(known_words.metadata) = 'object' THEN known_words.metadata ELSE '{}'::jsonb END || CASE WHEN jsonb_typeof(EXCLUDED.metadata) = 'object' THEN EXCLUDED.metadata ELSE '{}'::jsonb END"),
				"status":      "known",
			}),
		}).Create(&changed).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func dictionaryImportGradeLevels(tx *gorm.DB, language string, entries []VocabularyImportEntry, existing map[string]models.KnownWord) (map[string]int, error) {
	levels := make(map[string]int)
	if language != "ja" {
		return levels, nil
	}

	lemmas := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.JLPTLevel != nil {
			continue
		}
		if row, found := existing[entry.Lemma]; found && row.GradeLevel != nil {
			continue
		}
		lemmas = append(lemmas, entry.Lemma)
	}
	if len(lemmas) == 0 {
		return levels, nil
	}

	var rows []struct {
		Kanji     string
		Hiragana  string
		JLPTLevel *int
	}
	if err := tx.Model(&models.JapaneseDictionary{}).
		Select("kanji, hiragana, jlpt_level").
		Where("kanji IN ? OR hiragana IN ?", lemmas, lemmas).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.JLPTLevel == nil {
			continue
		}
		if row.Kanji != "" {
			levels[row.Kanji] = max(levels[row.Kanji], *row.JLPTLevel)
		}
		if row.Hiragana != "" {
			levels[row.Hiragana] = max(levels[row.Hiragana], *row.JLPTLevel)
		}
	}
	return levels, nil
}

func importedKnownWord(userID uint, language string, entry VocabularyImportEntry, existing models.KnownWord, found bool, inferredGrade int) (models.KnownWord, bool, error) {
	var metadata map[string]any
	if found && len(existing.Metadata) > 0 {
		if err := json.Unmarshal(existing.Metadata, &metadata); err != nil {
			return models.KnownWord{}, false, fmt.Errorf("failed to decode metadata for %s: %w", entry.Lemma, err)
		}
	}
	metadataPatch := make(map[string]string)
	metadataChanged := false
	if entry.Hiragana != "" {
		metadataPatch["hiragana"] = entry.Hiragana
		metadataChanged = metadata["hiragana"] != entry.Hiragana
	}
	if entry.Meaning != "" {
		metadataPatch["meaning"] = entry.Meaning
		metadataChanged = metadataChanged || metadata["meaning"] != entry.Meaning
	}

	var gradeLevel *int
	switch {
	case entry.JLPTLevel != nil:
		level := *entry.JLPTLevel
		gradeLevel = &level
	case found && existing.GradeLevel != nil:
		level := *existing.GradeLevel
		gradeLevel = &level
	case inferredGrade != 0:
		level := inferredGrade
		gradeLevel = &level
	}

	var metadataJSON []byte
	if len(metadataPatch) > 0 {
		encoded, err := json.Marshal(metadataPatch)
		if err != nil {
			return models.KnownWord{}, false, err
		}
		metadataJSON = encoded
	}

	changed := !found || existing.Status != "known" || !equalOptionalInt(existing.GradeLevel, gradeLevel) || metadataChanged
	return models.KnownWord{
		UserID:     userID,
		Language:   language,
		Lemma:      entry.Lemma,
		GradeLevel: gradeLevel,
		Status:     "known",
		Metadata:   metadataJSON,
	}, changed, nil
}

func equalOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
