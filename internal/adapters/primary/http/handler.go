package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// Handler handles HTTP requests for Ask Moto
type Handler struct {
	chatService      ports.ChatService
	feedbackService  ports.FeedbackService
	analyticsService ports.AnalyticsService
}

// NewHandler creates a new HTTP handler
func NewHandler(
	chatService ports.ChatService,
	feedbackService ports.FeedbackService,
	analyticsService ports.AnalyticsService,
) *Handler {
	return &Handler{
		chatService:      chatService,
		feedbackService:  feedbackService,
		analyticsService: analyticsService,
	}
}

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Message string `json:"message" example:"What is Moto?"`
	UserID  string `json:"user_id,omitempty" example:"user-123"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response   string            `json:"response" example:"Moto is a ride-hailing app that helps riders request rides, choose pricing modes, and helps drivers receive ride orders."`
	AnswerType domain.AnswerType `json:"answer_type" example:"kb_answer"`
	Confidence float64           `json:"confidence" example:"0.95"`
	ResponseID string            `json:"response_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// FeedbackRequest represents feedback from the user
type FeedbackRequest struct {
	ResponseID string `json:"response_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Rating     int    `json:"rating" example:"2" enums:"1,2"`
	Comment    string `json:"comment,omitempty" example:"Very helpful answer!"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Status string `json:"status" example:"ok"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status string `json:"status" example:"healthy"`
}

// HandleChat handles the /api/chat endpoint
// @Summary Send a message to Ask Moto
// @Description Submit a question and receive an AI-generated response based on the Moto knowledge base
// @Tags Chat
// @Accept json
// @Produce json
// @Param request body ChatRequest true "Chat request"
// @Success 200 {object} ChatResponse "Successful response"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Router /api/chat [post]
func (h *Handler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Message is required"})
		return
	}

	// Create question
	question := &domain.Question{
		ID:        uuid.New().String(),
		Text:      req.Message,
		UserID:    req.UserID,
		Timestamp: time.Now(),
	}

	// Process through chat service
	response := h.chatService.ProcessQuestion(question)

	// Return response
	chatResp := ChatResponse{
		Response:   response.Text,
		AnswerType: response.AnswerType,
		Confidence: response.Confidence,
		ResponseID: response.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

// HandleFeedback handles the /api/feedback endpoint
// @Summary Submit feedback on a response
// @Description Submit user feedback (thumbs up/down) on a previous response
// @Tags Feedback
// @Accept json
// @Produce json
// @Param request body FeedbackRequest true "Feedback request"
// @Success 200 {object} SuccessResponse "Feedback submitted successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/feedback [post]
func (h *Handler) HandleFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.ResponseID == "" || (req.Rating != 1 && req.Rating != 2) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Valid response_id and rating (1 or 2) required"})
		return
	}

	feedback := &domain.Feedback{
		ID:         uuid.New().String(),
		ResponseID: req.ResponseID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	}

	if err := h.feedbackService.SubmitFeedback(feedback); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to save feedback"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{Status: "ok"})
}

// HandleStats handles the /api/stats endpoint
// @Summary Get analytics statistics
// @Description Retrieve aggregated statistics about queries and feedback
// @Tags Analytics
// @Produce json
// @Success 200 {object} domain.Stats "Statistics retrieved successfully"
// @Failure 405 {object} ErrorResponse "Method not allowed"
// @Router /api/stats [get]
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Method not allowed"})
		return
	}

	stats := h.analyticsService.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HandleHealth handles the /health endpoint
// @Summary Health check
// @Description Check if the service is running and healthy
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Router /health [get]
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{Status: "healthy"})
}

// SetupRoutes configures all HTTP routes
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/chat", h.HandleChat)
	mux.HandleFunc("/api/feedback", h.HandleFeedback)
	mux.HandleFunc("/api/stats", h.HandleStats)
	mux.HandleFunc("/health", h.HandleHealth)
}
