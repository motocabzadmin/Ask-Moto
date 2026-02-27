package ports

import "github.com/moto/ask-moto/internal/core/domain"

// KnowledgeBaseRepository defines the output port for KB operations
// This is implemented by secondary adapters (file store, database, etc.)
type KnowledgeBaseRepository interface {
	// LoadFromDirectory loads all markdown files from a directory
	LoadFromDirectory(path string) error
	// GetDocument retrieves a document by ID
	GetDocument(id string) (*domain.Document, error)
	// GetAllDocuments returns all documents
	GetAllDocuments() []*domain.Document
	// GetDocumentsByModule returns documents filtered by module
	GetDocumentsByModule(module string) []*domain.Document
	// GetAllChunks returns all chunks
	GetAllChunks() []*domain.Chunk
	// AddDocument adds a document to the store
	AddDocument(doc *domain.Document) error
}

// Retriever defines the output port for KB retrieval operations
type Retriever interface {
	// Search finds relevant chunks for a query
	Search(query string, topK int) []*domain.ScoredChunk
	// SearchByModule finds relevant chunks within a specific module
	SearchByModule(query string, module string, topK int) []*domain.ScoredChunk
}

// Logger defines the output port for analytics logging
type Logger interface {
	// LogQuery logs a query and response
	LogQuery(log *domain.QueryLog) error
	// LogFeedback logs user feedback
	LogFeedback(feedback *domain.Feedback) error
	// GetQueryLogs returns all query logs
	GetQueryLogs() []*domain.QueryLog
	// GetFeedback returns all feedback
	GetFeedback() []*domain.Feedback
	// GetUnknownQuestions returns questions that weren't answered
	GetUnknownQuestions() []*domain.QueryLog
	// GetStats returns analytics statistics
	GetStats() domain.Stats
}

// IntentClassifier defines the output port for intent classification
type IntentClassifier interface {
	// Classify detects the intent of a question
	Classify(question string) domain.Intent
}

// LLMClient defines the output port for LLM operations
type LLMClient interface {
	// GenerateResponse generates a response using the LLM with given context
	GenerateResponse(question string, context string, systemPrompt string) (string, error)
}

// Embedder defines the output port for text embedding operations
type Embedder interface {
	// Embed generates a vector embedding for a single text
	Embed(text string) ([]float32, error)
	// EmbedBatch generates vector embeddings for multiple texts
	EmbedBatch(texts []string) ([][]float32, error)
}
