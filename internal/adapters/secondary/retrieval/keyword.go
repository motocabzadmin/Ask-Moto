package retrieval

import (
	"sort"
	"strings"

	"github.com/moto/ask-moto/internal/config"
	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// KeywordRetriever implements the Retriever port using keyword-based retrieval
type KeywordRetriever struct {
	store ports.KnowledgeBaseRepository
	cfg   *config.Config
}

// NewKeywordRetriever creates a new keyword-based retriever
func NewKeywordRetriever(store ports.KnowledgeBaseRepository, cfg *config.Config) *KeywordRetriever {
	return &KeywordRetriever{
		store: store,
		cfg:   cfg,
	}
}

// Search finds relevant chunks using keyword matching
func (r *KeywordRetriever) Search(query string, topK int) []*domain.ScoredChunk {
	chunks := r.store.GetAllChunks()
	return r.scoreAndRank(query, chunks, topK)
}

// SearchByModule finds relevant chunks within a specific module
func (r *KeywordRetriever) SearchByModule(query string, module string, topK int) []*domain.ScoredChunk {
	allChunks := r.store.GetAllChunks()
	var moduleChunks []*domain.Chunk

	for _, chunk := range allChunks {
		if chunk.Module == module {
			moduleChunks = append(moduleChunks, chunk)
		}
	}

	return r.scoreAndRank(query, moduleChunks, topK)
}

// scoreAndRank scores chunks and returns top-k results
func (r *KeywordRetriever) scoreAndRank(query string, chunks []*domain.Chunk, topK int) []*domain.ScoredChunk {
	queryLower := strings.ToLower(query)
	queryTokens := tokenize(queryLower)

	var scored []*domain.ScoredChunk

	for _, chunk := range chunks {
		score := r.calculateScore(queryTokens, chunk)
		if score > 0 {
			scored = append(scored, &domain.ScoredChunk{
				Chunk: chunk,
				Score: score,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top-k
	if len(scored) > topK {
		scored = scored[:topK]
	}

	return scored
}

// calculateScore calculates relevance score for a chunk
func (r *KeywordRetriever) calculateScore(queryTokens []string, chunk *domain.Chunk) float64 {
	chunkLower := strings.ToLower(chunk.Content)
	chunkTokens := tokenize(chunkLower)

	// Create a set of chunk tokens for faster lookup
	chunkTokenSet := make(map[string]int)
	for _, token := range chunkTokens {
		chunkTokenSet[token]++
	}

	var score float64

	// Score based on token matches
	for _, qToken := range queryTokens {
		// Exact match
		if count, ok := chunkTokenSet[qToken]; ok {
			score += float64(count) * 2.0
		}

		// Partial match (substring)
		if strings.Contains(chunkLower, qToken) {
			score += 1.0
		}
	}

	// Boost for important keywords using config values
	importantTerms := map[string]float64{
		"auto":          r.cfg.KeywordBoostHigh,
		"instant":       r.cfg.KeywordBoostHigh,
		"flex":          r.cfg.KeywordBoostHigh,
		"bidding":       r.cfg.KeywordBoostHigh,
		"pricing":       r.cfg.KeywordBoostMedium,
		"launch":        r.cfg.KeywordBoostMedium,
		"customization": r.cfg.KeywordBoostMedium,
		"saved":         r.cfg.KeywordBoostMedium,
		"places":        r.cfg.KeywordBoostMedium,
		"moto":          r.cfg.KeywordBoostLow,
		"ride":          r.cfg.KeywordBoostLow,
		"driver":        r.cfg.KeywordBoostLow,
	}

	for term, boost := range importantTerms {
		for _, qToken := range queryTokens {
			if qToken == term && strings.Contains(chunkLower, term) {
				score += boost
			}
		}
	}

	// Normalize by chunk length to avoid favoring very long chunks
	if len(chunkTokens) > 0 {
		score = score / (1 + float64(len(chunkTokens))/100)
	}

	return score
}

// tokenize splits text into lowercase tokens
func tokenize(text string) []string {
	// Remove punctuation and split by whitespace
	replacer := strings.NewReplacer(
		"?", " ",
		"!", " ",
		".", " ",
		",", " ",
		":", " ",
		";", " ",
		"'", " ",
		"\"", " ",
		"(", " ",
		")", " ",
		"-", " ",
	)

	cleaned := replacer.Replace(text)
	tokens := strings.Fields(cleaned)

	// Filter out very short tokens
	var filtered []string
	for _, t := range tokens {
		if len(t) >= 2 {
			filtered = append(filtered, t)
		}
	}

	return filtered
}
