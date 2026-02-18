package services

import (
	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// FeedbackServiceImpl implements the FeedbackService input port
type FeedbackServiceImpl struct {
	logger ports.Logger
}

// NewFeedbackService creates a new feedback service
func NewFeedbackService(logger ports.Logger) *FeedbackServiceImpl {
	return &FeedbackServiceImpl{
		logger: logger,
	}
}

// SubmitFeedback records user feedback on a response
func (s *FeedbackServiceImpl) SubmitFeedback(feedback *domain.Feedback) error {
	return s.logger.LogFeedback(feedback)
}
