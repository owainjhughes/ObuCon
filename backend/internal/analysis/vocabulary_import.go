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

const maxVocabularyImportBodyBytes = 4 << 20

type VocabularyImportEntry struct {
	Lemma     string `json:"lemma" binding:"required"`
	Hiragana  string `json:"hiragana,omitempty"`
	Meaning   string `json:"meaning,omitempty"`
	JLPTLevel *int   `json:"jlpt_level,omitempty" binding:"omitempty,min=1,max=5"`
}

type VocabularyImportResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

func normalizeVocabularyImportEntries(entries []VocabularyImportEntry) ([]VocabularyImportEntry, error) {
	normalized := make([]VocabularyImportEntry, 0, len(entries))
	indexByLemma := make(map[string]int, len(entries))
	for i, entry := range entries {
		entry.Lemma = strings.TrimSpace(entry.Lemma)
		entry.Hiragana = strings.TrimSpace(entry.Hiragana)
		entry.Meaning = strings.TrimSpace(entry.Meaning)
		if entry.Lemma == "" {
			return nil, fmt.Errorf("entry %d: lemma cannot be empty", i+1)
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
	return s.repo.ImportVocabulary(ctx, userID, language, entries)
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
		existingByLemma := make(map[string]*models.KnownWord, len(existingRows))
		for i := range existingRows {
			existingByLemma[existingRows[i].Lemma] = &existingRows[i]
		}

		needsGrade := make([]string, 0, len(entries))
		for _, entry := range entries {
			existing := existingByLemma[entry.Lemma]
			if entry.JLPTLevel != nil || (existing != nil && existing.GradeLevel != nil) {
				continue
			}
			needsGrade = append(needsGrade, entry.Lemma)
		}
		gradeLevels, err := (&Repository{db: tx}).GetDictionaryGradeLevels(ctx, language, needsGrade)
		if err != nil {
			return err
		}

		changed := make([]models.KnownWord, 0, len(entries))
		for _, entry := range entries {
			existing := existingByLemma[entry.Lemma]
			word, didChange, err := importedKnownWord(userID, language, entry, existing, gradeLevels[entry.Lemma])
			if err != nil {
				return err
			}
			switch {
			case existing == nil:
				result.Added++
			case didChange:
				result.Updated++
			default:
				result.Skipped++
			}
			if existing == nil || didChange {
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
				"metadata":    gorm.Expr("COALESCE(known_words.metadata, '{}'::jsonb) || COALESCE(EXCLUDED.metadata, '{}'::jsonb)"),
				"status":      "known",
			}),
		}).Create(&changed).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func importedKnownWord(userID uint, language string, entry VocabularyImportEntry, existing *models.KnownWord, inferredGrade int) (models.KnownWord, bool, error) {
	var metadata map[string]any
	if existing != nil && len(existing.Metadata) > 0 {
		if err := json.Unmarshal(existing.Metadata, &metadata); err != nil {
			return models.KnownWord{}, false, fmt.Errorf("failed to decode metadata for %s: %w", entry.Lemma, err)
		}
	}

	metadataPatch := map[string]string{}
	if entry.Hiragana != "" {
		metadataPatch["hiragana"] = entry.Hiragana
	}
	if entry.Meaning != "" {
		metadataPatch["meaning"] = entry.Meaning
	}
	metadataChanged := false
	for key, value := range metadataPatch {
		if metadata[key] != value {
			metadataChanged = true
		}
	}

	var gradeLevel *int
	switch {
	case entry.JLPTLevel != nil:
		gradeLevel = entry.JLPTLevel
	case existing != nil && existing.GradeLevel != nil:
		gradeLevel = existing.GradeLevel
	case inferredGrade != 0:
		gradeLevel = &inferredGrade
	}

	var metadataJSON []byte
	if len(metadataPatch) > 0 {
		encoded, err := json.Marshal(metadataPatch)
		if err != nil {
			return models.KnownWord{}, false, err
		}
		metadataJSON = encoded
	}

	changed := existing == nil ||
		existing.Status != "known" ||
		!sameOptionalInt(existing.GradeLevel, gradeLevel) ||
		metadataChanged
	return models.KnownWord{
		UserID:     userID,
		Language:   language,
		Lemma:      entry.Lemma,
		GradeLevel: gradeLevel,
		Status:     "known",
		Metadata:   metadataJSON,
	}, changed, nil
}

func sameOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
