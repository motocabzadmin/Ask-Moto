package retrieval

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/moto/ask-moto/internal/config"
	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// VectorRetriever implements the Retriever port using pgvector semantic search
type VectorRetriever struct {
	pool     *pgxpool.Pool
	embedder ports.Embedder
	cfg      *config.Config
}

// NewVectorRetriever creates a new vector-based retriever
func NewVectorRetriever(pool *pgxpool.Pool, embedder ports.Embedder, cfg *config.Config) *VectorRetriever {
	return &VectorRetriever{
		pool:     pool,
		embedder: embedder,
		cfg:      cfg,
	}
}

// Search finds relevant chunks using vector similarity search
func (r *VectorRetriever) Search(query string, topK int) []*domain.ScoredChunk {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate embedding for the query
	queryEmbedding, err := r.embedder.Embed(query)
	if err != nil {
		log.Printf("Failed to generate query embedding: %v", err)
		return nil
	}

	// Perform vector similarity search using cosine distance
	// The <=> operator calculates cosine distance, lower = more similar
	rows, err := r.pool.Query(ctx,
		`SELECT id, document_id, content, module, start_index, end_index,
		        1 - (embedding <=> $1) as similarity
		 FROM chunks
		 WHERE embedding IS NOT NULL
		 ORDER BY embedding <=> $1
		 LIMIT $2`,
		pgvector.NewVector(queryEmbedding), topK)
	if err != nil {
		log.Printf("Failed to search chunks: %v", err)
		return nil
	}
	defer rows.Close()

	var results []*domain.ScoredChunk
	for rows.Next() {
		var chunk domain.Chunk
		var score float64
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &chunk.Module, &chunk.StartIndex, &chunk.EndIndex, &score); err != nil {
			log.Printf("Failed to scan chunk: %v", err)
			continue
		}

		// Only include results above minimum score threshold
		if score >= r.cfg.MinRetrievalScore {
			results = append(results, &domain.ScoredChunk{
				Chunk: &chunk,
				Score: score,
			})
		}
	}

	return results
}

// SearchByModule finds relevant chunks within a specific module using vector similarity
func (r *VectorRetriever) SearchByModule(query string, module string, topK int) []*domain.ScoredChunk {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate embedding for the query
	queryEmbedding, err := r.embedder.Embed(query)
	if err != nil {
		log.Printf("Failed to generate query embedding: %v", err)
		return nil
	}

	// Perform vector similarity search with module filter
	rows, err := r.pool.Query(ctx,
		`SELECT id, document_id, content, module, start_index, end_index,
		        1 - (embedding <=> $1) as similarity
		 FROM chunks
		 WHERE embedding IS NOT NULL AND module = $3
		 ORDER BY embedding <=> $1
		 LIMIT $2`,
		pgvector.NewVector(queryEmbedding), topK, module)
	if err != nil {
		log.Printf("Failed to search chunks by module: %v", err)
		return nil
	}
	defer rows.Close()

	var results []*domain.ScoredChunk
	for rows.Next() {
		var chunk domain.Chunk
		var score float64
		if err := rows.Scan(&chunk.ID, &chunk.DocumentID, &chunk.Content, &chunk.Module, &chunk.StartIndex, &chunk.EndIndex, &score); err != nil {
			log.Printf("Failed to scan chunk: %v", err)
			continue
		}

		// Only include results above minimum score threshold
		if score >= r.cfg.MinRetrievalScore {
			results = append(results, &domain.ScoredChunk{
				Chunk: &chunk,
				Score: score,
			})
		}
	}

	return results
}
