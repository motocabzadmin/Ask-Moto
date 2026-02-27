package knowledgebase

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// VectorStore implements the KnowledgeBaseRepository port using PostgreSQL with pgvector
type VectorStore struct {
	pool     *pgxpool.Pool
	embedder ports.Embedder
	version  string
}

// NewVectorStore creates a new vector-based KB store
func NewVectorStore(pool *pgxpool.Pool, embedder ports.Embedder, version string) *VectorStore {
	return &VectorStore{
		pool:     pool,
		embedder: embedder,
		version:  version,
	}
}

// LoadFromDirectory loads all markdown files from a directory, generates embeddings, and stores in PostgreSQL
func (s *VectorStore) LoadFromDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		doc, err := s.loadMarkdownFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}

		// Check if document already exists and is up to date
		existing, _ := s.GetDocument(doc.ID)
		if existing != nil {
			// Skip if content hasn't changed
			if existing.Content == doc.Content {
				log.Printf("Skipping unchanged document: %s", doc.ID)
				continue
			}
			// Delete old chunks for this document
			if err := s.deleteDocumentChunks(doc.ID); err != nil {
				log.Printf("Warning: failed to delete old chunks for %s: %v", doc.ID, err)
			}
		}

		// Insert or update document
		if err := s.upsertDocument(doc); err != nil {
			return fmt.Errorf("failed to store document %s: %w", doc.ID, err)
		}

		// Chunk and embed the document
		chunks := s.chunkDocument(doc)
		if err := s.storeChunksWithEmbeddings(chunks); err != nil {
			return fmt.Errorf("failed to store chunks for %s: %w", doc.ID, err)
		}

		log.Printf("Loaded document: %s (%d chunks)", doc.ID, len(chunks))
	}

	return nil
}

// loadMarkdownFile reads and parses a markdown file into a Document
func (s *VectorStore) loadMarkdownFile(path string) (*domain.Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var content strings.Builder
	var title string
	var module string

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Extract title from first H1
		if lineNum == 1 && strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Extract module from front matter or filename
		if strings.HasPrefix(line, "module:") {
			module = strings.TrimSpace(strings.TrimPrefix(line, "module:"))
			continue
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Derive module from filename if not in content
	if module == "" {
		base := filepath.Base(path)
		module = strings.TrimSuffix(base, ".md")
	}

	// Generate ID from filename
	id := strings.TrimSuffix(filepath.Base(path), ".md")

	return &domain.Document{
		ID:        id,
		Title:     title,
		Content:   content.String(),
		Module:    module,
		Version:   s.version,
		UpdatedAt: time.Now(),
	}, nil
}

// chunkDocument splits a document into searchable chunks
func (s *VectorStore) chunkDocument(doc *domain.Document) []*domain.Chunk {
	chunks := make([]*domain.Chunk, 0)
	sections := splitIntoSections(doc.Content)

	for i, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		chunk := &domain.Chunk{
			ID:         fmt.Sprintf("%s-chunk-%d", doc.ID, i),
			DocumentID: doc.ID,
			Content:    section,
			Module:     doc.Module,
			StartIndex: 0,
			EndIndex:   len(section),
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

// storeChunksWithEmbeddings generates embeddings and stores chunks in the database
func (s *VectorStore) storeChunksWithEmbeddings(chunks []*domain.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Extract text content for batch embedding
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	// Generate embeddings in batch
	embeddings, err := s.embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Store chunks with embeddings
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for i, chunk := range chunks {
		embedding := pgvector.NewVector(embeddings[i])
		_, err := s.pool.Exec(ctx,
			`INSERT INTO chunks (id, document_id, content, module, embedding, start_index, end_index, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			 ON CONFLICT (id) DO UPDATE SET
			 content = EXCLUDED.content,
			 embedding = EXCLUDED.embedding`,
			chunk.ID, chunk.DocumentID, chunk.Content, chunk.Module, embedding, chunk.StartIndex, chunk.EndIndex)
		if err != nil {
			return fmt.Errorf("failed to insert chunk %s: %w", chunk.ID, err)
		}
	}

	return nil
}

// upsertDocument inserts or updates a document in the database
func (s *VectorStore) upsertDocument(doc *domain.Document) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO documents (id, title, content, module, version, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (id) DO UPDATE SET
		 title = EXCLUDED.title,
		 content = EXCLUDED.content,
		 module = EXCLUDED.module,
		 version = EXCLUDED.version,
		 updated_at = NOW()`,
		doc.ID, doc.Title, doc.Content, doc.Module, doc.Version)
	if err != nil {
		return fmt.Errorf("failed to upsert document: %w", err)
	}

	return nil
}

// deleteDocumentChunks removes all chunks for a document
func (s *VectorStore) deleteDocumentChunks(documentID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, "DELETE FROM chunks WHERE document_id = $1", documentID)
	return err
}

// GetDocument retrieves a document by ID
func (s *VectorStore) GetDocument(id string) (*domain.Document, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doc domain.Document
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, content, module, version, updated_at FROM documents WHERE id = $1`, id).
		Scan(&doc.ID, &doc.Title, &doc.Content, &doc.Module, &doc.Version, &doc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("document not found: %s", id)
	}

	return &doc, nil
}

// GetAllDocuments returns all documents
func (s *VectorStore) GetAllDocuments() []*domain.Document {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id, title, content, module, version, updated_at FROM documents`)
	if err != nil {
		log.Printf("Failed to query documents: %v", err)
		return nil
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		var doc domain.Document
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.Module, &doc.Version, &doc.UpdatedAt); err != nil {
			log.Printf("Failed to scan document: %v", err)
			continue
		}
		docs = append(docs, &doc)
	}

	return docs
}

// GetDocumentsByModule returns documents filtered by module
func (s *VectorStore) GetDocumentsByModule(module string) []*domain.Document {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id, title, content, module, version, updated_at FROM documents WHERE module = $1`, module)
	if err != nil {
		log.Printf("Failed to query documents by module: %v", err)
		return nil
	}
	defer rows.Close()

	var docs []*domain.Document
	for rows.Next() {
		var doc domain.Document
		if err := rows.Scan(&doc.ID, &doc.Title, &doc.Content, &doc.Module, &doc.Version, &doc.UpdatedAt); err != nil {
			log.Printf("Failed to scan document: %v", err)
			continue
		}
		docs = append(docs, &doc)
	}

	return docs
}

// GetAllChunks returns all chunks
func (s *VectorStore) GetAllChunks() []*domain.Chunk {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id, document_id, content, module, start_index, end_index FROM chunks`)
	if err != nil {
		log.Printf("Failed to query chunks: %v", err)
		return nil
	}
	defer rows.Close()

	var chunks []*domain.Chunk
	for rows.Next() {
		var chunk domain.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &chunk.Module, &chunk.StartIndex, &chunk.EndIndex); err != nil {
			log.Printf("Failed to scan chunk: %v", err)
			continue
		}
		chunks = append(chunks, &chunk)
	}

	return chunks
}

// AddDocument adds a document to the store with embedding generation
func (s *VectorStore) AddDocument(doc *domain.Document) error {
	// Store the document
	if err := s.upsertDocument(doc); err != nil {
		return err
	}

	// Chunk and embed
	chunks := s.chunkDocument(doc)
	return s.storeChunksWithEmbeddings(chunks)
}
