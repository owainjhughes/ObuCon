package analysis

import (
	"context"
	"obucon/internal/models"

	"gorm.io/gorm"
)

type AnalysisTokenRepository interface {
	Create(ctx context.Context, token *models.AnalysisToken) error
	BulkCreate(ctx context.Context, tokens []models.AnalysisToken) error
	GetByAnalysisID(ctx context.Context, analysisID uint) ([]models.AnalysisToken, error)
	GetUnknownByAnalysisID(ctx context.Context, analysisID uint) ([]models.AnalysisToken, error)
	Delete(ctx context.Context, id uint) error
}

type analysisTokenRepository struct {
	db *gorm.DB
}

func NewAnalysisTokenRepository(db *gorm.DB) AnalysisTokenRepository {
	return &analysisTokenRepository{db: db}
}

func (r *analysisTokenRepository) Create(ctx context.Context, token *models.AnalysisToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *analysisTokenRepository) BulkCreate(ctx context.Context, tokens []models.AnalysisToken) error {
	return r.db.WithContext(ctx).CreateInBatches(tokens, 100).Error
}

func (r *analysisTokenRepository) GetByAnalysisID(ctx context.Context, analysisID uint) ([]models.AnalysisToken, error) {
	var tokens []models.AnalysisToken
	err := r.db.WithContext(ctx).Where("analysis_id = ?", analysisID).Find(&tokens).Error
	return tokens, err
}

func (r *analysisTokenRepository) GetUnknownByAnalysisID(ctx context.Context, analysisID uint) ([]models.AnalysisToken, error) {
	var tokens []models.AnalysisToken
	err := r.db.WithContext(ctx).Where("analysis_id = ? AND is_known = ?", analysisID, false).Order("grade_level ASC").Find(&tokens).Error
	return tokens, err
}

func (r *analysisTokenRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AnalysisToken{}, id).Error
}
