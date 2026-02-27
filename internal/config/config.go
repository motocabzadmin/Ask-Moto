package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// defaultSystemPrompt is the default system prompt for the LLM
const defaultSystemPrompt = `You are Ask Moto, a helpful assistant for the Moto ride-hailing app.
You answer user questions about Moto's features, pricing modes, and functionality.

Guidelines:
- Be concise and helpful
- Only answer based on the provided knowledge base context
- If the context doesn't contain the answer, say you don't have that information
- Do not make up features or information not in the context
- Keep responses friendly but professional`

// Config holds all application configuration
type Config struct {
	// Server settings
	Port int

	// JWT Authentication settings
	JWTSecret string

	// Knowledge Base settings
	KBPath    string
	KBVersion string

	// Logging settings
	LogsPath string

	// Retrieval settings
	TopK              int
	MinRetrievalScore float64

	// Intent classification settings
	IntentMinConfidence   float64
	IntentMatchThreshold  float64
	SafeDefaultConfidence float64

	// Question matching settings
	QuestionOverlapThreshold   float64
	SentenceRelevanceThreshold float64

	// Keyword boost scores
	KeywordBoostHigh   float64
	KeywordBoostMedium float64
	KeywordBoostLow    float64

	// OpenAI settings
	OpenAIAPIKey    string
	OpenAIModel     string
	OpenAIMaxTokens int
	LLMEnabled      bool
	LLMSystemPrompt string

	// Database settings (pgvector)
	DatabaseURL     string
	DatabaseEnabled bool

	// Embedding settings
	EmbeddingModel     string
	EmbeddingDimension int
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := loadEnvFile(); err != nil {
		// Ignore error if file doesn't exist
	}

	cfg := &Config{
		// Server settings
		Port: getEnvInt("PORT", 8080),

		// JWT Authentication settings (compatible with motocabz identity service)
		JWTSecret: getEnvString("JWT_SECRET", "tp54XJqd7sb7vw8dQXgRZcHdv3k3+YI7fUgaPdZStY8="),

		// Knowledge Base settings
		KBPath:    getEnvString("KB_PATH", "data/kb"),
		KBVersion: getEnvString("KB_VERSION", "v1"),

		// Logging settings
		LogsPath: getEnvString("LOGS_PATH", "data/logs"),

		// Retrieval settings
		TopK:              getEnvInt("TOP_K", 5),
		MinRetrievalScore: getEnvFloat("MIN_RETRIEVAL_SCORE", 1.0),

		// Intent classification settings
		IntentMinConfidence:   getEnvFloat("INTENT_MIN_CONFIDENCE", 0.3),
		IntentMatchThreshold:  getEnvFloat("INTENT_MATCH_THRESHOLD", 0.7),
		SafeDefaultConfidence: getEnvFloat("SAFE_DEFAULT_CONFIDENCE", 0.7),

		// Question matching settings
		QuestionOverlapThreshold:   getEnvFloat("QUESTION_OVERLAP_THRESHOLD", 0.7),
		SentenceRelevanceThreshold: getEnvFloat("SENTENCE_RELEVANCE_THRESHOLD", 0.4),

		// Keyword boost scores
		KeywordBoostHigh:   getEnvFloat("KEYWORD_BOOST_HIGH", 3.0),
		KeywordBoostMedium: getEnvFloat("KEYWORD_BOOST_MEDIUM", 2.5),
		KeywordBoostLow:    getEnvFloat("KEYWORD_BOOST_LOW", 1.5),

		// OpenAI settings
		OpenAIAPIKey:    getEnvString("OPENAI_API_KEY", ""),
		OpenAIModel:     getEnvString("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIMaxTokens: getEnvInt("OPENAI_MAX_TOKENS", 500),
		LLMEnabled:      getEnvBool("LLM_ENABLED", false),
		LLMSystemPrompt: getEnvString("LLM_SYSTEM_PROMPT", defaultSystemPrompt),

		// Database settings (pgvector)
		DatabaseURL:     getEnvString("DATABASE_URL", "postgres://askmoto:askmoto@localhost:5432/askmoto"),
		DatabaseEnabled: getEnvBool("DATABASE_ENABLED", false),

		// Embedding settings
		EmbeddingModel:     getEnvString("EMBEDDING_MODEL", "text-embedding-3-small"),
		EmbeddingDimension: getEnvInt("EMBEDDING_DIMENSION", 1536),
	}

	return cfg, nil
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() error {
	// Try to find .env file in current directory or parent directories
	envPaths := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	var envFile string
	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			envFile = path
			break
		}
	}

	if envFile == "" {
		return nil // No .env file found, use defaults
	}

	file, err := os.Open(envFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// getEnvString returns a string environment variable or default
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt returns an int environment variable or default
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// getEnvFloat returns a float64 environment variable or default
func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultVal
}

// getEnvBool returns a bool environment variable or default
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}
