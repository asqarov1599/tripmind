package main

import (
	"log"
	"os"
	"strings"
	"time"
	"tripmind/database"
	"tripmind/handlers"
	"tripmind/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file (ignored in production where env vars are set directly)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables")
	}

	// Initialize database
	database.InitDB()

	// Initialize Amadeus service
	services.InitAmadeus()

	// Initialize AI service
	services.InitAI()

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Trusted proxies (Railway sits behind a proxy)
	r.SetTrustedProxies([]string{"0.0.0.0/0"})

	// CORS — allow configured frontend origins
	frontendURLs := os.Getenv("FRONTEND_URL")
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if frontendURLs != "" {
		for _, u := range strings.Split(frontendURLs, ",") {
			u = strings.TrimSpace(u)
			if u != "" {
				allowedOrigins = append(allowedOrigins, u)
			}
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthHandler)
		api.POST("/search", handlers.SearchHandler)
		api.POST("/generate", handlers.GenerateHandler)
		api.GET("/download/:id", handlers.DownloadHandler)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 TripMind backend starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
