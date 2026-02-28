package auth

import (
	"context"
	"fmt"
	"obucon/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	fmt.Print("Auth Repository NewRepository Function Reached\n")
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, user *models.User) error {
	fmt.Print("Auth Repository Create Function Reached\n")
	result := r.db.WithContext(ctx).Create(user)
	return result.Error
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	fmt.Print("Auth Repository GetByID Function Reached\n")
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	return &user, err
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	fmt.Print("Auth Repository GetByEmail Function Reached\n")
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	fmt.Print("Auth Repository GetByUsername Function Reached\n")
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *Repository) Update(ctx context.Context, user *models.User) error {
	fmt.Print("Auth Repository Update Function Reached\n")
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	fmt.Print("Auth Repository Delete Function Reached\n")
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}
