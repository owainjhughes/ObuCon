package analysis

import (
	"context"
	"database/sql"
	"obucon/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertKnownWord(ctx context.Context, knownWord *models.KnownWord) error {
	if err := r.populateKnownWordGradeLevel(ctx, knownWord); err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "language"}, {Name: "lemma"}},
			DoUpdates: clause.AssignmentColumns([]string{"grade_level", "status"}),
		}).
		Create(knownWord).Error
}

func (r *Repository) populateKnownWordGradeLevel(ctx context.Context, knownWord *models.KnownWord) error {
	if knownWord == nil || knownWord.GradeLevel != nil || knownWord.Lemma == "" {
		return nil
	}

	switch knownWord.Language {
	case "ja":
		var jlptLevel sql.NullInt32
		err := r.db.WithContext(ctx).
			Model(&models.JapaneseDictionary{}).
			Select("jlpt_level").
			Where("kanji = ? OR hiragana = ?", knownWord.Lemma, knownWord.Lemma).
			Order("jlpt_level DESC").
			Limit(1).
			Scan(&jlptLevel).Error
		if err != nil {
			return err
		}

		if jlptLevel.Valid {
			level := int(jlptLevel.Int32)
			knownWord.GradeLevel = &level
		}
	}

	return nil
}

func (r *Repository) GetKnownLemmas(ctx context.Context, userID uint, language string, lemmas []string) (map[string]bool, error) {
	known := make(map[string]bool)
	if len(lemmas) == 0 {
		return known, nil
	}

	var rows []models.KnownWord
	err := r.db.WithContext(ctx).
		Select("lemma").
		Where("user_id = ? AND language = ? AND lemma IN ?", userID, language, lemmas).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		known[row.Lemma] = true
	}

	return known, nil
}

// expands the given lemmas by including other forms (kanji/hiragana variants) from the dictionary
func (r *Repository) GetKnownLemmasWithDictionaryVariants(ctx context.Context, userID uint, language string, lemmas []string) (map[string]bool, error) {
	// Start with the direct known lemmas.
	known, err := r.GetKnownLemmas(ctx, userID, language, lemmas)
	if err != nil {
		return nil, err
	}

	// if we already have known entries for all lemmas, skip
	if len(known) == len(lemmas) {
		return known, nil
	}

	variantMap := make(map[string]map[string]struct{}, len(lemmas))
	for _, l := range lemmas {
		variantMap[l] = map[string]struct{}{l: {}}
	}

	var dictRows []struct {
		Kanji    string `gorm:"column:kanji"`
		Hiragana string `gorm:"column:hiragana"`
	}
	err = r.db.WithContext(ctx).
		Model(&models.JapaneseDictionary{}).
		Select("kanji, hiragana").
		Where("kanji IN ? OR hiragana IN ?", lemmas, lemmas).
		Find(&dictRows).Error
	if err != nil {
		return nil, err
	}

	variantSet := make(map[string]struct{})
	for _, row := range dictRows {
		if row.Kanji == "" || row.Hiragana == "" {
			continue
		}

		// If the row matches any input lemma, add both forms as variants.
		if _, ok := variantMap[row.Kanji]; ok {
			variantMap[row.Kanji][row.Hiragana] = struct{}{}
		}
		if _, ok := variantMap[row.Hiragana]; ok {
			variantMap[row.Hiragana][row.Kanji] = struct{}{}
		}

		variantSet[row.Kanji] = struct{}{}
		variantSet[row.Hiragana] = struct{}{}
	}

	// Collect variants that are not already known.
	variants := make([]string, 0, len(variantSet))
	for v := range variantSet {
		if !known[v] {
			variants = append(variants, v)
		}
	}

	if len(variants) == 0 {
		return known, nil
	}

	variantKnown, err := r.GetKnownLemmas(ctx, userID, language, variants)
	if err != nil {
		return nil, err
	}

	// Mark any known variant as known for all related lemmas.
	for lemma, variants := range variantMap {
		for v := range variants {
			if variantKnown[v] {
				known[lemma] = true
				break
			}
		}
	}

	// Also include direct variant matches (e.g. if the user knows the kanji form directly).
	for v := range variantKnown {
		known[v] = true
	}

	return known, nil
}

type VocabEntry struct {
	Lemma      string `json:"lemma"`
	GradeLevel *int   `json:"grade_level"`
	Meaning    string `json:"meaning"`
}

func (r *Repository) ListKnownWordsWithMeaning(ctx context.Context, userID uint, language string) ([]VocabEntry, error) {
	var entries []VocabEntry

	switch language {
	case "ja":
		err := r.db.WithContext(ctx).
			Table("known_words").
			Select("known_words.lemma, known_words.grade_level, coalesce(nullif(known_words.metadata->>'meaning', ''), japanese_dictionary.meaning, '') AS meaning").
			Joins("LEFT JOIN japanese_dictionary ON known_words.lemma = japanese_dictionary.kanji OR known_words.lemma = japanese_dictionary.hiragana").
			Where("known_words.user_id = ? AND known_words.language = ?", userID, language).
			Order("known_words.created_at desc").
			Scan(&entries).Error
		if err != nil {
			return nil, err
		}
	default:
		// Fallback: just return known words without meaning
		var rows []models.KnownWord
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND language = ?", userID, language).
			Order("created_at desc").
			Find(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			entries = append(entries, VocabEntry{Lemma: row.Lemma, GradeLevel: row.GradeLevel})
		}
	}

	return entries, nil
}

func (r *Repository) BulkAddKnownWordsByJLPT(ctx context.Context, userID uint, language string, jlptLevel int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(
		`INSERT INTO known_words (user_id, language, lemma, grade_level, status, created_at)
		 SELECT ?, ?, CASE WHEN kanji <> '' THEN kanji ELSE hiragana END, ?, 'known', CURRENT_TIMESTAMP
		 FROM japanese_dictionary
		 WHERE jlpt_level = ?
		 ON CONFLICT (user_id, language, lemma) DO NOTHING`,
		userID, language, jlptLevel, jlptLevel,
	)
	return res.RowsAffected, res.Error
}

func (r *Repository) UpdateKnownWord(ctx context.Context, userID uint, language, lemma, meaning string, jlptLevel int) error {
	res := r.db.WithContext(ctx).Exec(
		`UPDATE known_words
		 SET grade_level = ?,
		     metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{meaning}', to_jsonb(?::text), true)
		 WHERE user_id = ? AND language = ? AND lemma = ?`,
		jlptLevel, meaning, userID, language, lemma,
	)

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *Repository) RemoveKnownWord(ctx context.Context, userID uint, language, lemma string) error {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND language = ? AND lemma = ?", userID, language, lemma).
		Delete(&models.KnownWord{})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

type ReviewWord struct {
	Lemma     string `json:"lemma"`
	Hiragana  string `json:"hiragana"`
	Meaning   string `json:"meaning"`
	JLPTLevel *int   `json:"jlpt_level"`
}

func (r *Repository) GetDictionaryEntries(ctx context.Context, language string, lemmas []string) ([]ReviewWord, error) {
	if len(lemmas) == 0 {
		return nil, nil
	}

	switch language {
	case "ja":
		type dictionaryRow struct {
			Kanji     string `gorm:"column:kanji"`
			Hiragana  string `gorm:"column:hiragana"`
			Meaning   string `gorm:"column:meaning"`
			JLPTLevel *int   `gorm:"column:jlpt_level"`
		}

		var rows []dictionaryRow
		err := r.db.WithContext(ctx).
			Model(&models.JapaneseDictionary{}).
			Select("kanji, hiragana, meaning, jlpt_level").
			Where("(kanji IN ? OR hiragana IN ?) AND meaning != ''", lemmas, lemmas).
			Order("jlpt_level DESC NULLS LAST").
			Find(&rows).Error
		if err != nil {
			return nil, err
		}

		lemmaSet := make(map[string]struct{}, len(lemmas))
		for _, l := range lemmas {
			lemmaSet[l] = struct{}{}
		}

		seen := make(map[string]struct{}, len(lemmas))
		result := make([]ReviewWord, 0, len(lemmas))
		for _, row := range rows {
			var matchedLemma string
			if _, ok := lemmaSet[row.Kanji]; ok && row.Kanji != "" {
				matchedLemma = row.Kanji
			} else if _, ok := lemmaSet[row.Hiragana]; ok {
				matchedLemma = row.Hiragana
			} else {
				continue
			}

			if _, found := seen[matchedLemma]; found {
				continue
			}
			seen[matchedLemma] = struct{}{}
			result = append(result, ReviewWord{
				Lemma:     matchedLemma,
				Hiragana:  row.Hiragana,
				Meaning:   row.Meaning,
				JLPTLevel: row.JLPTLevel,
			})
		}

		return result, nil
	}

	return nil, nil
}

func (r *Repository) GetDictionaryGradeLevels(ctx context.Context, language string, lemmas []string) (map[string]int, error) {
	levels := make(map[string]int)
	if len(lemmas) == 0 {
		return levels, nil
	}

	switch language {
	case "ja":
		type dictionaryRow struct {
			Kanji     string `gorm:"column:kanji"`
			Hiragana  string `gorm:"column:hiragana"`
			JLPTLevel *int   `gorm:"column:jlpt_level"`
		}

		var rows []dictionaryRow
		err := r.db.WithContext(ctx).
			Model(&models.JapaneseDictionary{}).
			Select("kanji, hiragana, jlpt_level").
			Where("jlpt_level IS NOT NULL AND (kanji IN ? OR hiragana IN ?)", lemmas, lemmas).
			Find(&rows).Error
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			if row.JLPTLevel == nil {
				continue
			}

			level := *row.JLPTLevel

			if row.Kanji != "" {
				if existing, ok := levels[row.Kanji]; !ok || level > existing {
					levels[row.Kanji] = level
				}
			}

			if row.Hiragana != "" {
				if existing, ok := levels[row.Hiragana]; !ok || level > existing {
					levels[row.Hiragana] = level
				}
			}
		}

	}

	return levels, nil
}
