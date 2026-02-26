package auth

import (
	"context"
	"fmt"
	"obucon/internal/models"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user database operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	result := r.db.WithContext(ctx).Create(user)
	fmt.Print("Creating User: ", user.Email, "\n")
	return result.Error
}

func (r *userRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	fmt.Print("Getting User by ID: ", id, "\n")
	return &user, err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	fmt.Print("Getting User by Email: ", email, "\n")
	return &user, err
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	fmt.Print("Getting User by Username: ", username, "\n")
	return &user, err
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	fmt.Print("Updating User: ", user.Email, "\n")
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
	fmt.Print("Deleting User with ID: ", id, "\n")
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}
