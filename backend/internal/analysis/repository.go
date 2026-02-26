package analysis

import (
	"context"
	"obucon/internal/models"

	"gorm.io/gorm"
)

type AnalysisRepository interface {
	Create(ctx context.Context, analysis *models.Analysis) error
	GetByID(ctx context.Context, id uint) (*models.Analysis, error)
	GetByUserID(ctx context.Context, userID uint) ([]models.Analysis, error)
	GetByTextHash(ctx context.Context, userID uint, textHash string) (*models.Analysis, error)
	Delete(ctx context.Context, id uint) error
}

type analysisRepository struct {
	db *gorm.DB
}

func NewAnalysisRepository(db *gorm.DB) AnalysisRepository {
	return &analysisRepository{db: db}
}

// Creates a new text analysis
func (r *analysisRepository) Create(ctx context.Context, analysis *models.Analysis) error {
	return r.db.WithContext(ctx).Create(analysis).Error
}

func (r *analysisRepository) GetByID(ctx context.Context, id uint) (*models.Analysis, error) {
	var analysis models.Analysis
	err := r.db.WithContext(ctx).First(&analysis, id).Error
	return &analysis, err
}

func (r *analysisRepository) GetByUserID(ctx context.Context, userID uint) ([]models.Analysis, error) {
	var analyses []models.Analysis
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&analyses).Error
	return analyses, err
}

func (r *analysisRepository) GetByTextHash(ctx context.Context, userID uint, textHash string) (*models.Analysis, error) {
	var analysis models.Analysis
	err := r.db.WithContext(ctx).Where("user_id = ? AND text_hash = ?", userID, textHash).First(&analysis).Error
	return &analysis, err
}

func (r *analysisRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Analysis{}, id).Error
}
