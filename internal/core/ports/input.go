package ports

import "github.com/moto/ask-moto/internal/core/domain"

// ChatService defines the input port for chat operations
// This is the primary use case interface that primary adapters (HTTP handlers) call
type ChatService interface {
	// ProcessQuestion processes a user question and returns a response
	ProcessQuestion(question *domain.Question) *domain.Response
}

// FeedbackService defines the input port for feedback operations
type FeedbackService interface {
	// SubmitFeedback records user feedback on a response
	SubmitFeedback(feedback *domain.Feedback) error
}

// AnalyticsService defines the input port for analytics operations
type AnalyticsService interface {
	// GetStats returns analytics statistics
	GetStats() domain.Stats
	// GetUnknownQuestions returns questions that weren't answered
	GetUnknownQuestions() []*domain.QueryLog
}
