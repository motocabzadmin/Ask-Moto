package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/moto/ask-moto/internal/core/domain"
)

// FileLogger implements the Logger port with file-based storage
type FileLogger struct {
	mu           sync.RWMutex
	queryLogs    []*domain.QueryLog
	feedback     []*domain.Feedback
	logFile      string
	feedbackFile string
}

// NewFileLogger creates a new file-based logger
func NewFileLogger(logDir string) (*FileLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	return &FileLogger{
		queryLogs:    make([]*domain.QueryLog, 0),
		feedback:     make([]*domain.Feedback, 0),
		logFile:      filepath.Join(logDir, "queries.jsonl"),
		feedbackFile: filepath.Join(logDir, "feedback.jsonl"),
	}, nil
}

// LogQuery logs a query and response
func (l *FileLogger) LogQuery(log *domain.QueryLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	log.Timestamp = time.Now()
	l.queryLogs = append(l.queryLogs, log)

	// Append to file
	return l.appendToFile(l.logFile, log)
}

// LogFeedback logs user feedback
func (l *FileLogger) LogFeedback(feedback *domain.Feedback) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	feedback.Timestamp = time.Now()
	l.feedback = append(l.feedback, feedback)

	return l.appendToFile(l.feedbackFile, feedback)
}

// appendToFile appends a JSON line to a file
func (l *FileLogger) appendToFile(filename string, data interface{}) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(encoded) + "\n")
	return err
}

// GetQueryLogs returns all query logs
func (l *FileLogger) GetQueryLogs() []*domain.QueryLog {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.queryLogs
}

// GetFeedback returns all feedback
func (l *FileLogger) GetFeedback() []*domain.Feedback {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.feedback
}

// GetUnknownQuestions returns questions that weren't answered
func (l *FileLogger) GetUnknownQuestions() []*domain.QueryLog {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var unknown []*domain.QueryLog
	for _, log := range l.queryLogs {
		if log.AnswerType == domain.AnswerTypeUnknown {
			unknown = append(unknown, log)
		}
	}
	return unknown
}

// GetDownvotedResponses returns responses that received negative feedback
func (l *FileLogger) GetDownvotedResponses() []*domain.Feedback {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var downvoted []*domain.Feedback
	for _, fb := range l.feedback {
		if fb.Rating == 1 { // thumbs down
			downvoted = append(downvoted, fb)
		}
	}
	return downvoted
}

// GetStats returns analytics statistics
func (l *FileLogger) GetStats() domain.Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := domain.Stats{
		TotalQueries:  len(l.queryLogs),
		TotalFeedback: len(l.feedback),
	}

	for _, log := range l.queryLogs {
		if log.AnswerType == domain.AnswerTypeUnknown {
			stats.UnknownQueries++
		} else {
			stats.AnsweredQueries++
		}
	}

	if stats.TotalQueries > 0 {
		stats.AnswerRate = float64(stats.AnsweredQueries) / float64(stats.TotalQueries)
	}

	for _, fb := range l.feedback {
		if fb.Rating == 2 {
			stats.PositiveFeedback++
		} else {
			stats.NegativeFeedback++
		}
	}

	return stats
}
