package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	httphandler "github.com/moto/ask-moto/internal/adapters/primary/http"
	"github.com/moto/ask-moto/internal/adapters/primary/http/middleware"
	"github.com/moto/ask-moto/internal/adapters/secondary/database"
	"github.com/moto/ask-moto/internal/adapters/secondary/embedding"
	"github.com/moto/ask-moto/internal/adapters/secondary/intent"
	"github.com/moto/ask-moto/internal/adapters/secondary/knowledgebase"
	"github.com/moto/ask-moto/internal/adapters/secondary/llm"
	"github.com/moto/ask-moto/internal/adapters/secondary/logging"
	"github.com/moto/ask-moto/internal/adapters/secondary/retrieval"
	"github.com/moto/ask-moto/internal/config"
	"github.com/moto/ask-moto/internal/core/ports"
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

	// 1. Knowledge Base Repository and Retriever
	var kbStore ports.KnowledgeBaseRepository
	var retriever ports.Retriever

	if cfg.DatabaseEnabled {
		// Use pgvector-based vector store for semantic search
		log.Println("Database enabled - using pgvector for semantic search")

		// Initialize database connection pool
		pool, err := database.NewPostgresPool(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Run migrations
		log.Println("Running database migrations...")
		if err := database.RunMigrations(pool, "migrations"); err != nil {
			log.Printf("Warning: Migration failed (may already be applied): %v", err)
		}

		// Initialize embedder
		if cfg.OpenAIAPIKey == "" {
			log.Fatal("OPENAI_API_KEY is required when DATABASE_ENABLED=true")
		}
		embedder := embedding.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.EmbeddingModel)

		// Initialize vector store
		vectorStore := knowledgebase.NewVectorStore(pool, embedder, cfg.KBVersion)

		// Check if database is empty and needs seeding
		empty, err := database.IsEmpty(pool)
		if err != nil {
			log.Printf("Warning: Could not check if database is empty: %v", err)
		}

		if empty {
			log.Println("Database is empty, loading knowledge base from:", cfg.KBPath)
			if err := vectorStore.LoadFromDirectory(cfg.KBPath); err != nil {
				if _, statErr := os.Stat(cfg.KBPath); os.IsNotExist(statErr) {
					log.Printf("KB directory does not exist: %s. Creating...", cfg.KBPath)
					if err := os.MkdirAll(cfg.KBPath, 0755); err != nil {
						log.Fatalf("Failed to create KB directory: %v", err)
					}
					log.Println("Empty KB created. Please add markdown files to:", cfg.KBPath)
				} else {
					log.Printf("Warning: Failed to load KB: %v", err)
				}
			}
		} else {
			log.Println("Database already seeded, skipping KB load")
		}

		kbStore = vectorStore
		retriever = retrieval.NewVectorRetriever(pool, embedder, cfg)
		log.Printf("Using vector-based semantic search (model: %s)", cfg.EmbeddingModel)
	} else {
		// Use file-based keyword store (original behavior)
		log.Println("Loading knowledge base from:", cfg.KBPath)
		fileStore := knowledgebase.NewFileStore(cfg.KBVersion)
		if err := fileStore.LoadFromDirectory(cfg.KBPath); err != nil {
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
		kbStore = fileStore
		retriever = retrieval.NewKeywordRetriever(fileStore, cfg)
		log.Println("Using keyword-based retrieval")
	}

	docs := kbStore.GetAllDocuments()
	chunks := kbStore.GetAllChunks()
	log.Printf("Loaded %d documents with %d chunks", len(docs), len(chunks))

	// 3. Intent Classifier
	classifier := intent.NewClassifier(cfg)

	// 4. Logger
	logger, err := logging.NewFileLogger(cfg.LogsPath)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 5. LLM Client (optional - enabled via config)
	var llmClient ports.LLMClient
	if cfg.LLMEnabled && cfg.OpenAIAPIKey != "" {
		llmClient = llm.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIMaxTokens)
		log.Printf("LLM enabled: using %s model", cfg.OpenAIModel)
	} else if cfg.LLMEnabled {
		log.Println("Warning: LLM_ENABLED is true but OPENAI_API_KEY is not set")
	}

	// Initialize application services (core business logic)
	chatService := services.NewChatService(retriever, classifier, logger, llmClient, cfg)
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
