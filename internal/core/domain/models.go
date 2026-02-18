package domain

import "time"

// AnswerType represents the type of response Ask Moto provides
type AnswerType string

const (
	// AnswerTypeKB is a pure knowledge response from the KB
	AnswerTypeKB AnswerType = "kb_answer"
	// AnswerTypeActionGuidance provides in-app navigation guidance
	AnswerTypeActionGuidance AnswerType = "action_guidance"
	// AnswerTypeUnknown indicates the KB has no coverage
	AnswerTypeUnknown AnswerType = "unknown"
)

// Document represents a knowledge base document
type Document struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Module    string    `json:"module"` // overview, pricing_modes, customization, saved_places, faq
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Chunk represents a searchable chunk of text from a document
type Chunk struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	Content    string `json:"content"`
	Module     string `json:"module"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
}

// Question represents a user question to Ask Moto
type Question struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Response represents Ask Moto's response
type Response struct {
	ID              string     `json:"id"`
	QuestionID      string     `json:"question_id"`
	Text            string     `json:"text"`
	AnswerType      AnswerType `json:"answer_type"`
	RetrievedChunks []Chunk    `json:"retrieved_chunks,omitempty"`
	Confidence      float64    `json:"confidence"`
	Timestamp       time.Time  `json:"timestamp"`
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatSession represents a chat conversation
type ChatSession struct {
	ID        string        `json:"id"`
	UserID    string        `json:"user_id,omitempty"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Intent represents a detected user intent for structured handling
type Intent struct {
	Name       string            `json:"name"`
	Confidence float64           `json:"confidence"`
	Params     map[string]string `json:"params,omitempty"`
}

// KnownIntent constants for rule-based classification
const (
	IntentLaunchDate     = "launch_date"
	IntentLaunchLocation = "launch_location"
	IntentWhatIsMoto     = "what_is_moto"
	IntentPricingInstant = "pricing_instant"
	IntentPricingFlex    = "pricing_flex"
	IntentPricingAuto    = "pricing_auto"
	IntentAutoBidding    = "auto_bidding"
	IntentCustomization  = "customization"
	IntentSavedPlaces    = "saved_places"
	IntentRequestRide    = "request_ride"
	IntentUnknown        = "unknown"
)

// SafeDefault contains canonical safe responses
type SafeDefault struct {
	Intent   string `json:"intent"`
	Response string `json:"response"`
}

// QueryLog represents a log entry for analytics
type QueryLog struct {
	ID              string     `json:"id"`
	QuestionID      string     `json:"question_id"`
	QuestionText    string     `json:"question_text"`
	DetectedIntent  string     `json:"detected_intent"`
	RetrievedChunks []string   `json:"retrieved_chunk_ids"`
	ResponseText    string     `json:"response_text"`
	AnswerType      AnswerType `json:"answer_type"`
	Timestamp       time.Time  `json:"timestamp"`
}

// Feedback represents user feedback on a response
type Feedback struct {
	ID         string    `json:"id"`
	ResponseID string    `json:"response_id"`
	Rating     int       `json:"rating"` // 1 = thumbs down, 2 = thumbs up
	Comment    string    `json:"comment,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// ScoredChunk represents a chunk with its relevance score
type ScoredChunk struct {
	Chunk *Chunk
	Score float64
}

// QAPair represents a question-answer pair extracted from content
type QAPair struct {
	Question string
	Answer   string
}

// Stats represents analytics statistics
type Stats struct {
	TotalQueries     int     `json:"total_queries"`
	AnsweredQueries  int     `json:"answered_queries"`
	UnknownQueries   int     `json:"unknown_queries"`
	AnswerRate       float64 `json:"answer_rate"`
	TotalFeedback    int     `json:"total_feedback"`
	PositiveFeedback int     `json:"positive_feedback"`
	NegativeFeedback int     `json:"negative_feedback"`
}
