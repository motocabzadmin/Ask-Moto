package services

import (
	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// AnalyticsServiceImpl implements the AnalyticsService input port
type AnalyticsServiceImpl struct {
	logger ports.Logger
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(logger ports.Logger) *AnalyticsServiceImpl {
	return &AnalyticsServiceImpl{
		logger: logger,
	}
}

// GetStats returns analytics statistics
func (s *AnalyticsServiceImpl) GetStats() domain.Stats {
	return s.logger.GetStats()
}

// GetUnknownQuestions returns questions that weren't answered
func (s *AnalyticsServiceImpl) GetUnknownQuestions() []*domain.QueryLog {
	return s.logger.GetUnknownQuestions()
}
