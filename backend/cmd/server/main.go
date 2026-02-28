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

func main() {
	log.Print("Main Function Reached")

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
	authHandler := auth.NewAuthHandler(authService)

	tokenizer, err := ja.NewTokenizer()
	if err != nil {
		log.Fatalf("Failed to initialize tokenizer: %v", err)
	}

	analysisService := analysis.NewService(tokenizer)
	analysisHandler := analysis.NewAnalysisHandler(analysisService)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000"},
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
	{
		protected.GET("/auth/me", authHandler.GetMe)
		protected.POST("/analyze", analysisHandler.AnalyzeText)
	}

	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
