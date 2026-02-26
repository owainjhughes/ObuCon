package vocab

import (
	"context"
	"obucon/internal/models"

	"gorm.io/gorm"
)

type VocabularyRepository interface {
	Create(ctx context.Context, item *models.VocabularyItem) error
	GetByID(ctx context.Context, id uint) (*models.VocabularyItem, error)
	GetByUserID(ctx context.Context, userID uint) ([]models.VocabularyItem, error)
	GetByUserAndLanguage(ctx context.Context, userID uint, language string) ([]models.VocabularyItem, error)
	GetByUserAndLemma(ctx context.Context, userID uint, lemma string, language string) (*models.VocabularyItem, error)
	Update(ctx context.Context, item *models.VocabularyItem) error
	Delete(ctx context.Context, id uint) error
	BulkCreate(ctx context.Context, items []models.VocabularyItem) error
}

type vocabularyRepository struct {
	db *gorm.DB
}

func NewVocabularyRepository(db *gorm.DB) VocabularyRepository {
	return &vocabularyRepository{db: db}
}

func (r *vocabularyRepository) Create(ctx context.Context, item *models.VocabularyItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *vocabularyRepository) GetByID(ctx context.Context, id uint) (*models.VocabularyItem, error) {
	var vocab models.VocabularyItem
	err := r.db.WithContext(ctx).First(&vocab, id).Error
	return &vocab, err
}

func (r *vocabularyRepository) GetByUserID(ctx context.Context, userID uint) ([]models.VocabularyItem, error) {
	var items []models.VocabularyItem
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("lemma ASC").Find(&items).Error
	return items, err
}

func (r *vocabularyRepository) GetByUserAndLanguage(ctx context.Context, userID uint, language string) ([]models.VocabularyItem, error) {
	var items []models.VocabularyItem
	err := r.db.WithContext(ctx).Where("user_id = ? AND language = ?", userID, language).Order("lemma ASC").Find(&items).Error
	return items, err
}

func (r *vocabularyRepository) GetByUserAndLemma(ctx context.Context, userID uint, lemma string, language string) (*models.VocabularyItem, error) {
	var item models.VocabularyItem
	err := r.db.WithContext(ctx).Where("user_id = ? AND lemma = ? AND language = ?", userID, lemma, language).First(&item).Error
	return &item, err
}

func (r *vocabularyRepository) Update(ctx context.Context, item *models.VocabularyItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *vocabularyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.VocabularyItem{}, id).Error
}

func (r *vocabularyRepository) BulkCreate(ctx context.Context, items []models.VocabularyItem) error {
	return r.db.WithContext(ctx).CreateInBatches(items, 100).Error
}
