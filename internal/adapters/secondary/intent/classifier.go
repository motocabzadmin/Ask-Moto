package intent

import (
	"strings"

	"github.com/moto/ask-moto/internal/config"
	"github.com/moto/ask-moto/internal/core/domain"
)

// Classifier implements the IntentClassifier port using rule-based classification
type Classifier struct {
	patterns map[string][]string
	cfg      *config.Config
}

// NewClassifier creates a new rule-based intent classifier
func NewClassifier(cfg *config.Config) *Classifier {
	c := &Classifier{
		patterns: make(map[string][]string),
		cfg:      cfg,
	}
	c.initPatterns()
	return c
}

// initPatterns sets up the pattern matching rules
func (c *Classifier) initPatterns() {
	// Launch date patterns
	c.patterns[domain.IntentLaunchDate] = []string{
		"when is moto launching",
		"when will moto launch",
		"launch date",
		"when does moto start",
		"when can i use moto",
		"release date",
		"when is the launch",
		"when launching",
	}

	// Launch location patterns
	c.patterns[domain.IntentLaunchLocation] = []string{
		"where is moto launching",
		"where will moto launch",
		"which city",
		"which cities",
		"available in",
		"launch location",
		"where available",
		"where is moto available",
		"addis ababa",
		"bole",
	}

	// What is Moto patterns
	c.patterns[domain.IntentWhatIsMoto] = []string{
		"what is moto",
		"what does moto do",
		"tell me about moto",
		"explain moto",
		"moto app",
		"what is this app",
	}

	// Instant pricing patterns
	c.patterns[domain.IntentPricingInstant] = []string{
		"what is instant",
		"instant mode",
		"instant pricing",
		"how does instant work",
		"fastest mode",
		"quick ride",
	}

	// Flex pricing patterns
	c.patterns[domain.IntentPricingFlex] = []string{
		"what is flex",
		"flex mode",
		"flex pricing",
		"negotiate",
		"negotiation",
		"counter offer",
		"accept decline",
	}

	// Auto pricing patterns
	c.patterns[domain.IntentPricingAuto] = []string{
		"what is auto",
		"auto mode",
		"auto pricing",
		"how does auto work",
		"price clearing",
		"step down",
		"timer",
		"30 seconds",
		"45 seconds",
		"automatic pricing",
		"calm",
	}

	// Auto bidding confusion patterns (must check for this separately)
	c.patterns[domain.IntentAutoBidding] = []string{
		"is auto bidding",
		"auto bidding",
		"auto bid",
		"drivers bid",
		"bidding war",
		"auction",
		"compete",
		"competition between drivers",
		"is it bidding",
	}

	// Customization patterns
	c.patterns[domain.IntentCustomization] = []string{
		"customization",
		"customize",
		"conversation",
		"quiet mode",
		"chat mode",
		"music",
		"radio",
		"notes",
		"driver notes",
		"special instructions",
		"preferences",
	}

	// Saved places patterns
	c.patterns[domain.IntentSavedPlaces] = []string{
		"saved places",
		"your places",
		"save address",
		"add new",
		"favorite places",
		"pin place",
		"edit place",
		"delete place",
		"home address",
		"work address",
	}

	// Request ride patterns
	c.patterns[domain.IntentRequestRide] = []string{
		"request a ride",
		"how to request",
		"book a ride",
		"order a ride",
		"get a ride",
		"request ride",
		"ride for someone else",
		"share trip",
		"track driver",
		"rate driver",
	}
}

// Classify detects the intent of a question
func (c *Classifier) Classify(question string) domain.Intent {
	questionLower := strings.ToLower(question)

	var bestIntent string
	var bestScore float64

	for intent, patterns := range c.patterns {
		score := c.scorePatterns(questionLower, patterns)
		if score > bestScore {
			bestScore = score
			bestIntent = intent
		}
	}

	// Require minimum confidence
	if bestScore < c.cfg.IntentMinConfidence {
		return domain.Intent{
			Name:       domain.IntentUnknown,
			Confidence: 0.0,
		}
	}

	return domain.Intent{
		Name:       bestIntent,
		Confidence: bestScore,
	}
}

// scorePatterns calculates how well the question matches the patterns
func (c *Classifier) scorePatterns(question string, patterns []string) float64 {
	var maxScore float64

	for _, pattern := range patterns {
		score := c.matchPattern(question, pattern)
		if score > maxScore {
			maxScore = score
		}
	}

	return maxScore
}

// matchPattern calculates the match score between question and pattern
func (c *Classifier) matchPattern(question, pattern string) float64 {
	// Exact substring match
	if strings.Contains(question, pattern) {
		return 1.0
	}

	// Token overlap
	qTokens := strings.Fields(question)
	pTokens := strings.Fields(pattern)

	if len(pTokens) == 0 {
		return 0
	}

	var matches int
	for _, pt := range pTokens {
		for _, qt := range qTokens {
			if qt == pt || strings.Contains(qt, pt) || strings.Contains(pt, qt) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(pTokens))
}
