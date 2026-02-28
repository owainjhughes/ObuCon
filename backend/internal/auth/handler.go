package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *Service
}

func NewAuthHandler(authService *Service) *AuthHandler {
	fmt.Print("Auth NewAuthHandler Function Reached\n")
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=4"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	Email string `json:"email"`
	ID    uint   `json:"id"`
}

const (
	authCookieName          = "auth_token"
	authCookieMaxAgeSeconds = 24 * 60 * 60
)

func setAuthCookie(c *gin.Context, token string) {
	fmt.Print("Auth setAuthCookie Function Reached\n")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authCookieName, token, authCookieMaxAgeSeconds, "/", "", false, true)
}

func clearAuthCookie(c *gin.Context) {
	fmt.Print("Auth clearAuthCookie Function Reached\n")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authCookieName, "", -1, "/", "", false, true)
}

func (h *AuthHandler) Register(c *gin.Context) {
	fmt.Print("Auth Register Handler Function Reached\n")

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"username": user.Username,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	fmt.Print("Auth Login Handler Function Reached\n")

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, userID, err := h.authService.LoginWithUserID(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	setAuthCookie(c, token)

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		Email: req.Email,
		ID:    userID,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	fmt.Print("Auth Logout Handler Function Reached\n")

	clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	fmt.Print("Auth GetMe Handler Function Reached\n")

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": userID,
	})
}

func AuthMiddleware(authService *Service) gin.HandlerFunc {
	fmt.Print("Auth Middleware Function Reached\n")

	return func(c *gin.Context) {
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else {
			cookieToken, err := c.Cookie(authCookieName)
			if err != nil || cookieToken == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
				c.Abort()
				return
			}
			tokenString = cookieToken
		}

		userID, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
