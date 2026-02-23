package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	httphandler "github.com/moto/ask-moto/internal/adapters/primary/http"
	"github.com/moto/ask-moto/internal/adapters/primary/http/middleware"
	"github.com/moto/ask-moto/internal/adapters/secondary/intent"
	"github.com/moto/ask-moto/internal/adapters/secondary/knowledgebase"
	"github.com/moto/ask-moto/internal/adapters/secondary/logging"
	"github.com/moto/ask-moto/internal/adapters/secondary/retrieval"
	"github.com/moto/ask-moto/internal/config"
	"github.com/moto/ask-moto/internal/core/services"
	"github.com/moto/ask-moto/internal/identity"

	_ "github.com/moto/ask-moto/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Ask Moto API
// @version 1.0
// @description A conversational Q&A service for the Moto ride-hailing app.
// @description Ask Moto answers user questions about Moto's features, pricing modes, and functionality using a knowledge base retrieval system.

// @contact.name Moto Team
// @contact.email support@moto.app

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @tag.name Chat
// @tag.description Conversational Q&A endpoints
// @tag.name Feedback
// @tag.description User feedback on responses
// @tag.name Analytics
// @tag.description Usage statistics and metrics
// @tag.name Health
// @tag.description Service health monitoring
func main() {
	// Load configuration from .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize secondary adapters (driven adapters)

	// 1. Knowledge Base Repository
	log.Println("Loading knowledge base from:", cfg.KBPath)
	kbStore := knowledgebase.NewFileStore(cfg.KBVersion)
	if err := kbStore.LoadFromDirectory(cfg.KBPath); err != nil {
		// Check if directory exists
		if _, statErr := os.Stat(cfg.KBPath); os.IsNotExist(statErr) {
			log.Printf("KB directory does not exist: %s. Creating with sample data...", cfg.KBPath)
			if err := os.MkdirAll(cfg.KBPath, 0755); err != nil {
				log.Fatalf("Failed to create KB directory: %v", err)
			}
			log.Println("Empty KB created. Please add markdown files to:", cfg.KBPath)
		} else {
			log.Printf("Warning: Failed to load KB: %v", err)
		}
	}

	docs := kbStore.GetAllDocuments()
	chunks := kbStore.GetAllChunks()
	log.Printf("Loaded %d documents with %d chunks", len(docs), len(chunks))

	// 2. Retriever
	retriever := retrieval.NewKeywordRetriever(kbStore, cfg)

	// 3. Intent Classifier
	classifier := intent.NewClassifier(cfg)

	// 4. Logger
	logger, err := logging.NewFileLogger(cfg.LogsPath)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize application services (core business logic)
	chatService := services.NewChatService(retriever, classifier, logger, cfg)
	feedbackService := services.NewFeedbackService(logger)
	analyticsService := services.NewAnalyticsService(logger)

	// Initialize primary adapters (driving adapters)
	handler := httphandler.NewHandler(chatService, feedbackService, analyticsService)

	// Initialize identity service for JWT token validation (compatible with motocabz)
	tokenService := identity.NewTokenService(cfg.JWTSecret)
	authMiddleware := middleware.AuthMiddleware(tokenService)

	// Setup HTTP routes with authentication middleware
	mux := http.NewServeMux()

	// Protected routes - require authentication (driver/rider token)
	mux.Handle("/api/chat", authMiddleware(http.HandlerFunc(handler.HandleChat)))
	mux.Handle("/api/feedback", authMiddleware(http.HandlerFunc(handler.HandleFeedback)))

	// Public routes - no authentication required
	mux.HandleFunc("/api/stats", handler.HandleStats)
	mux.HandleFunc("/health", handler.HandleHealth)

	// Swagger documentation
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting Ask Moto server on %s", addr)
	log.Println("Endpoints:")
	log.Println("  POST /api/chat       - Send a message to Ask Moto")
	log.Println("  POST /api/feedback   - Submit feedback on a response")
	log.Println("  GET  /api/stats      - View analytics statistics")
	log.Println("  GET  /health         - Health check")
	log.Println("  GET  /swagger/       - Swagger UI documentation")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
