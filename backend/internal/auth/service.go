package auth

import (
	"context"
	"errors"
	"fmt"
	"obucon/internal/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	userRepo  *Repository
	jwtSecret string
}

func NewService(userRepo *Repository, jwtSecret string) *Service {
	fmt.Print("Auth NewService Function Reached\n")
	return &Service{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) generateToken(userID uint, email string) (string, error) {
	fmt.Print("Auth generateToken Function Reached\n")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) Register(ctx context.Context, email, username, password string) (*models.User, error) {
	fmt.Print("Auth Register Function Reached\n")

	if email == "" {
		return nil, errors.New("email is required")
	}

	if len(username) < 3 || len(username) > 50 {
		return nil, errors.New("username must be between 3 and 50 characters")
	}

	if len(password) < 4 {
		return nil, errors.New("password must be at least 4 characters")
	}

	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already registered")
	}

	existingUser, err = s.userRepo.GetByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return nil, errors.New("username already taken")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *Service) LoginWithUserID(ctx context.Context, email, password string) (string, uint, error) {
	fmt.Print("Auth LoginWithUserID Function Reached\n")

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		return "", 0, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", 0, errors.New("invalid email or password")
	}

	tokenString, err := s.generateToken(user.ID, user.Email)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, user.ID, nil
}

func (s *Service) ValidateToken(tokenString string) (uint, error) {
	fmt.Print("Auth ValidateToken Function Reached\n")

	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return 0, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token claims")
	}

	if exp, ok := (*claims)["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return 0, errors.New("token expired")
		}
	}

	userID, ok := (*claims)["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid user_id in token")
	}

	return uint(userID), nil
}
