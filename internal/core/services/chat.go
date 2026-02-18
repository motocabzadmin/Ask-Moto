package services

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/moto/ask-moto/internal/config"
	"github.com/moto/ask-moto/internal/core/domain"
	"github.com/moto/ask-moto/internal/core/ports"
)

// SafeDefaults contains canonical responses that must be consistent
var SafeDefaults = map[string]string{
	domain.IntentLaunchDate: "Moto has not announced a public launch date yet.",

	domain.IntentLaunchLocation: "Moto plans to launch in Addis Ababa, starting with the Bole area, then expanding.",

	domain.IntentWhatIsMoto: "Moto is a ride-hailing app that helps riders request rides, choose pricing modes, and helps drivers receive ride orders.",

	domain.IntentPricingAuto: "Moto Auto is a calm, time-based price-clearing process. It runs for 30–45 seconds, with the price stepping down in fixed increments. When the timer ends, Moto clears the price by selecting a driver at the lowest price level that still has at least one eligible driver available, using clear tie-break rules when multiple drivers qualify at that same price.",

	domain.IntentAutoBidding: "No. Auto is not bidding. It is a calm, time-based price-clearing process.",

	domain.IntentPricingInstant: "Instant auto-assigns a driver quickly. The system assigns a driver automatically.",

	domain.IntentPricingFlex: "Flex lets riders negotiate fare with drivers. Riders can accept, counter, or decline.",

	domain.IntentCustomization: "Moto supports these customization options: conversation preference (No Preference, Quiet Mode, Chat Mode), music/sound preference (Quiet Mode, Radio/Music On), driver notes, ordering for someone else, and trip sharing.",

	domain.IntentSavedPlaces: "Saved Places lets riders store addresses they use often. Places can be pinned, edited, deleted, and favorited.",

	domain.IntentRequestRide: "To request a ride: Choose pickup and destination, choose a pricing mode, apply customization if needed, then get matched to a driver.",
}

// UnknownResponse is returned when KB has no coverage
const UnknownResponse = "This is not documented in the current Moto KB v1."

// ChatServiceImpl implements the ChatService input port
type ChatServiceImpl struct {
	retriever  ports.Retriever
	classifier ports.IntentClassifier
	logger     ports.Logger
	cfg        *config.Config
}

// NewChatService creates a new chat service
func NewChatService(
	retriever ports.Retriever,
	classifier ports.IntentClassifier,
	logger ports.Logger,
	cfg *config.Config,
) *ChatServiceImpl {
	return &ChatServiceImpl{
		retriever:  retriever,
		classifier: classifier,
		logger:     logger,
		cfg:        cfg,
	}
}

// ProcessQuestion processes a user question and returns a response
func (s *ChatServiceImpl) ProcessQuestion(question *domain.Question) *domain.Response {
	// Step 1: Classify intent
	intent := s.classifier.Classify(question.Text)

	// Step 2: Check for safe default (high-priority intents)
	if intent.Confidence >= s.cfg.SafeDefaultConfidence {
		if safeResponse, ok := SafeDefaults[intent.Name]; ok {
			response := &domain.Response{
				ID:         uuid.New().String(),
				QuestionID: question.ID,
				Text:       safeResponse,
				AnswerType: domain.AnswerTypeKB,
				Confidence: intent.Confidence,
				Timestamp:  time.Now(),
			}
			s.logQuery(question, response, intent.Name)
			return response
		}
	}

	// Step 3: Retrieve relevant chunks
	scoredChunks := s.retriever.Search(question.Text, s.cfg.TopK)

	// Step 4: If no relevant chunks found, return unknown
	if len(scoredChunks) == 0 || scoredChunks[0].Score < s.cfg.MinRetrievalScore {
		response := &domain.Response{
			ID:         uuid.New().String(),
			QuestionID: question.ID,
			Text:       UnknownResponse,
			AnswerType: domain.AnswerTypeUnknown,
			Confidence: 0.0,
			Timestamp:  time.Now(),
		}
		s.logQuery(question, response, intent.Name)
		return response
	}

	// Step 5: Try to find exact Q/A match in retrieved chunks
	for _, sc := range scoredChunks {
		qaPairs := ExtractQAPairs(sc.Chunk)
		for _, qa := range qaPairs {
			if s.questionMatches(question.Text, qa.Question) {
				response := &domain.Response{
					ID:              uuid.New().String(),
					QuestionID:      question.ID,
					Text:            qa.Answer,
					AnswerType:      domain.AnswerTypeKB,
					RetrievedChunks: []domain.Chunk{*sc.Chunk},
					Confidence:      0.9,
					Timestamp:       time.Now(),
				}
				s.logQuery(question, response, intent.Name)
				return response
			}
		}
	}

	// Step 6: Construct response from top chunk
	topChunk := scoredChunks[0]

	// Extract the most relevant sentence or section
	responseText := s.extractBestResponse(question.Text, topChunk.Chunk.Content)

	if responseText == "" {
		response := &domain.Response{
			ID:         uuid.New().String(),
			QuestionID: question.ID,
			Text:       UnknownResponse,
			AnswerType: domain.AnswerTypeUnknown,
			Confidence: 0.0,
			Timestamp:  time.Now(),
		}
		s.logQuery(question, response, intent.Name)
		return response
	}

	// Collect retrieved chunks
	var chunks []domain.Chunk
	for _, sc := range scoredChunks {
		chunks = append(chunks, *sc.Chunk)
	}

	response := &domain.Response{
		ID:              uuid.New().String(),
		QuestionID:      question.ID,
		Text:            responseText,
		AnswerType:      domain.AnswerTypeKB,
		RetrievedChunks: chunks,
		Confidence:      topChunk.Score / 10.0, // Normalize score
		Timestamp:       time.Now(),
	}
	s.logQuery(question, response, intent.Name)
	return response
}

// logQuery logs the query to the analytics logger
func (s *ChatServiceImpl) logQuery(question *domain.Question, response *domain.Response, intent string) {
	var chunkIDs []string
	for _, chunk := range response.RetrievedChunks {
		chunkIDs = append(chunkIDs, chunk.ID)
	}

	queryLog := &domain.QueryLog{
		ID:              uuid.New().String(),
		QuestionID:      question.ID,
		QuestionText:    question.Text,
		DetectedIntent:  intent,
		RetrievedChunks: chunkIDs,
		ResponseText:    response.Text,
		AnswerType:      response.AnswerType,
	}
	_ = s.logger.LogQuery(queryLog)
}

// questionMatches checks if user question matches a KB question
func (s *ChatServiceImpl) questionMatches(userQ, kbQ string) bool {
	userLower := strings.ToLower(strings.TrimSpace(userQ))
	kbLower := strings.ToLower(strings.TrimSpace(kbQ))

	// Remove trailing punctuation
	userLower = strings.TrimSuffix(userLower, "?")
	kbLower = strings.TrimSuffix(kbLower, "?")

	// Exact match
	if userLower == kbLower {
		return true
	}

	// High token overlap
	userTokens := strings.Fields(userLower)
	kbTokens := strings.Fields(kbLower)

	if len(kbTokens) == 0 {
		return false
	}

	var matches int
	for _, ut := range userTokens {
		for _, kt := range kbTokens {
			if ut == kt {
				matches++
				break
			}
		}
	}

	overlap := float64(matches) / float64(len(kbTokens))
	return overlap >= s.cfg.QuestionOverlapThreshold
}

// extractBestResponse extracts the most relevant response from chunk content
func (s *ChatServiceImpl) extractBestResponse(question, content string) string {
	questionLower := strings.ToLower(question)
	lines := strings.Split(content, "\n")

	// Look for Q/A format first
	for i, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.HasPrefix(lineLower, "q:") || strings.HasPrefix(lineLower, "**q:") {
			// Check if this Q matches
			if s.lineMatchesQuestion(questionLower, lineLower) {
				// Extract answer from same line or next line
				parts := strings.SplitN(line, "A:", 2)
				if len(parts) == 2 {
					answer := strings.TrimSpace(parts[1])
					answer = strings.TrimSuffix(answer, "**")
					return answer
				}
				// Check next line for answer
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(strings.ToLower(nextLine), "a:") {
						return strings.TrimSpace(strings.TrimPrefix(nextLine, "A:"))
					}
				}
			}
		}
	}

	// Look for relevant sentences
	sentences := splitSentences(content)
	for _, sent := range sentences {
		sentLower := strings.ToLower(sent)
		if s.sentenceRelevant(questionLower, sentLower) {
			return strings.TrimSpace(sent)
		}
	}

	// Return first non-empty, non-header line if nothing else matches
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && len(trimmed) > 20 {
			return trimmed
		}
	}

	return ""
}

// lineMatchesQuestion checks if a Q: line matches the user question
func (s *ChatServiceImpl) lineMatchesQuestion(question, qLine string) bool {
	// Extract just the question part
	qLine = strings.TrimPrefix(qLine, "q:")
	qLine = strings.TrimPrefix(qLine, "**q:")
	qLine = strings.TrimSuffix(qLine, "**")

	parts := strings.SplitN(qLine, "a:", 2)
	kbQuestion := strings.TrimSpace(parts[0])

	return s.questionMatches(question, kbQuestion)
}

// sentenceRelevant checks if a sentence is relevant to the question
func (s *ChatServiceImpl) sentenceRelevant(question, sentence string) bool {
	// Check for key terms overlap
	qTokens := strings.Fields(question)
	sTokens := strings.Fields(sentence)

	if len(qTokens) == 0 || len(sTokens) < 5 {
		return false
	}

	var matches int
	for _, qt := range qTokens {
		if len(qt) < 3 {
			continue
		}
		for _, st := range sTokens {
			if qt == st || strings.Contains(st, qt) {
				matches++
				break
			}
		}
	}

	return float64(matches)/float64(len(qTokens)) >= s.cfg.SentenceRelevanceThreshold
}

// splitSentences splits text into sentences
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			sent := strings.TrimSpace(current.String())
			if len(sent) > 10 {
				sentences = append(sentences, sent)
			}
			current.Reset()
		}
	}

	// Don't forget remaining text
	if current.Len() > 0 {
		sent := strings.TrimSpace(current.String())
		if len(sent) > 10 {
			sentences = append(sentences, sent)
		}
	}

	return sentences
}

// ExtractQAPairs extracts Q/A pairs from a chunk if it contains them
func ExtractQAPairs(chunk *domain.Chunk) []domain.QAPair {
	var pairs []domain.QAPair
	lines := strings.Split(chunk.Content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Q:") || strings.HasPrefix(line, "**Q:") {
			// Extract Q and A from the same line or subsequent lines
			parts := strings.SplitN(line, "A:", 2)
			if len(parts) == 2 {
				q := strings.TrimPrefix(parts[0], "Q:")
				q = strings.TrimPrefix(q, "**Q:")
				q = strings.TrimSuffix(q, "**")
				q = strings.TrimSpace(q)

				a := strings.TrimSpace(parts[1])
				a = strings.TrimSuffix(a, "**")

				pairs = append(pairs, domain.QAPair{
					Question: q,
					Answer:   a,
				})
			}
		}
	}

	return pairs
}
