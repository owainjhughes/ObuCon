package main

import (
	"log"
	"obucon/internal/analysis"
	"obucon/internal/auth"
	"obucon/internal/config"
	"obucon/internal/database"
	"obucon/internal/lang/ja"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func registerProtectedRoutes(group *gin.RouterGroup, authHandler *auth.AuthHandler, analysisHandler *analysis.AnalysisHandler) {
	group.GET("/auth/me", authHandler.GetMe)
	group.PUT("/auth/me", authHandler.UpdateMe)
	group.POST("/analyze", analysisHandler.AnalyzeText)
	group.POST("/analyze/file", analysisHandler.AnalyzeFile)
	group.GET("/vocab", analysisHandler.ListVocabulary)
	group.POST("/vocab/bulk", analysisHandler.BulkAddVocabulary)
	group.POST("/vocab/known", analysisHandler.AddKnownWord)
	group.PUT("/vocab/known", analysisHandler.UpdateKnownWord)
	group.DELETE("/vocab/known", analysisHandler.RemoveKnownWord)
	group.GET("/review/words", analysisHandler.GetReviewWords)
}

func main() {
	cfg := config.Load()

	if err := database.RunMigrations(cfg); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	userRepo := auth.NewRepository(db)
	authService := auth.NewService(userRepo, cfg.JWTSecret)
	authHandler := auth.NewAuthHandler(authService, cfg.CookieSecure)

	tokenizer, err := ja.NewTokenizer()
	if err != nil {
		log.Fatalf("Failed to initialize tokenizer: %v", err)
	}

	analysisRepo := analysis.NewRepository(db)
	analysisService := analysis.NewService(tokenizer, analysisRepo)
	analysisHandler := analysis.NewAnalysisHandler(analysisService)

	gin.SetMode(cfg.GinMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Failed to configure trusted proxies: %v", err)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/logout", authHandler.Logout)

	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware(authService))
	registerProtectedRoutes(protected, authHandler, analysisHandler)

	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
