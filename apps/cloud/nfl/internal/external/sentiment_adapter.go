package external

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// SentimentService defines the interface for generating sentiment analysis from fan reactions
type SentimentService interface {
	// AnalyzeSentiment generates a sentiment rating from Reddit comments
	// Returns a domain.SentimentRating with score, sentiment category, and key themes
	AnalyzeSentiment(source string, threadURL string, comments []string) (domain.SentimentRating, error)
}

// SentimentAdapter implements the SentimentService interface using OpenAI API
// This adapter converts OpenAI-specific responses to domain entities
type SentimentAdapter struct {
	apiKey string
	client *resty.Client
	prompt string // Custom prompt, uses default if empty
}

// NewSentimentAdapter creates a new sentiment adapter that implements SentimentService
func NewSentimentAdapter(apiKey string) SentimentService {
	return &SentimentAdapter{
		apiKey: apiKey,
		client: resty.New(),
		prompt: SentimentPrompt, // Use default prompt
	}
}

// NewCustomSentimentAdapter creates a new sentiment adapter with a custom prompt
func NewCustomSentimentAdapter(apiKey, customPrompt string) SentimentService {
	return &SentimentAdapter{
		apiKey: apiKey,
		client: resty.New(),
		prompt: customPrompt,
	}
}

// AnalyzeSentiment generates a sentiment rating from Reddit comments using OpenAI API
func (a *SentimentAdapter) AnalyzeSentiment(source string, threadURL string, comments []string) (domain.SentimentRating, error) {
	type Body struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	// Guard: require at least some comments to analyze
	if len(comments) == 0 {
		return domain.SentimentRating{}, fmt.Errorf("no comments provided for sentiment analysis")
	}

	// Convert comments to JSON for the API
	commentsJSON, err := json.Marshal(comments)
	if err != nil {
		log.Printf("Error marshaling comments: %v", err)
		return domain.SentimentRating{}, err
	}

	body := Body{
		Model:       "gpt-4o-mini",
		Temperature: 0.1,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: a.prompt + string(commentsJSON),
			},
		},
	}

	// Use environment variable for API URL in tests
	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/chat/completions"
	}

	log.Printf("=== OpenAI Sentiment Request ===")
	log.Printf("URL: %s", apiURL)
	log.Printf("Comment Count: %d", len(comments))
	log.Printf("Source: %s", source)
	log.Printf("Thread URL: %s", threadURL)
	log.Printf("================================")

	post, err := a.client.R().SetAuthToken(a.apiKey).SetBody(body).Post(apiURL)
	if err != nil {
		log.Printf("Error calling OpenAI API: %v", err)
		return domain.SentimentRating{}, err
	}

	// Log the response received from OpenAI
	log.Printf("=== OpenAI Sentiment Response ===")
	log.Printf("Status Code: %d", post.StatusCode())
	log.Printf("Response Headers: %v", post.Header())
	log.Printf("Raw Response Body: %s", post.String())
	log.Printf("=================================")

	type OuterResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// First check if there's an error in the response
	type ErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	var errorResponse ErrorResponse
	if errUnmarshal := json.Unmarshal([]byte(post.String()), &errorResponse); errUnmarshal == nil && errorResponse.Error.Message != "" {
		return domain.SentimentRating{}, fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
	}

	var outerJsonResponse OuterResponse
	err = json.Unmarshal([]byte(post.String()), &outerJsonResponse)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return domain.SentimentRating{}, err
	}

	if len(outerJsonResponse.Choices) == 0 {
		log.Printf("No choices in OpenAI response")
		return domain.SentimentRating{}, fmt.Errorf("no choices in OpenAI response")
	}

	jsonString := outerJsonResponse.Choices[0].Message.Content
	jsonString = strings.TrimPrefix(jsonString, "```json\n")
	jsonString = strings.TrimSuffix(jsonString, "\n```")
	jsonString = strings.ReplaceAll(jsonString, "\\n", "")
	jsonString = strings.ReplaceAll(jsonString, "\\\"", "\"")

	// Parse into temporary structure matching OpenAI response
	var tempResponse struct {
		Score      int      `json:"score"`
		Sentiment  string   `json:"sentiment"`
		Highlights []string `json:"highlights"`
	}

	err = json.Unmarshal([]byte(jsonString), &tempResponse)
	if err != nil {
		log.Printf("Error unmarshaling sentiment JSON: %v", err)
		return domain.SentimentRating{}, err
	}

	// Validate sentiment field
	validSentiments := map[string]bool{
		"excited":      true,
		"neutral":      true,
		"disappointed": true,
	}
	if !validSentiments[tempResponse.Sentiment] {
		log.Printf("Warning: invalid sentiment '%s', defaulting to 'neutral'", tempResponse.Sentiment)
		tempResponse.Sentiment = "neutral"
	}

	// Create domain SentimentRating entity
	rating := domain.SentimentRating{
		Source:       source,
		ThreadURL:    threadURL,
		CommentCount: len(comments),
		Score:        tempResponse.Score,
		Sentiment:    tempResponse.Sentiment,
		Highlights:   tempResponse.Highlights,
		GeneratedAt:  time.Now(),
	}

	log.Printf("Sentiment Score: %d", rating.Score)
	log.Printf("Sentiment: %s", rating.Sentiment)
	log.Printf("Highlights: %v", rating.Highlights)

	return rating, nil
}

// SentimentPrompt is the prompt for generating sentiment ratings from Reddit comments
const SentimentPrompt = `Analyze the provided Reddit comments from an NFL post-game thread and generate a sentiment score between 0 and 100, representing overall fan excitement and engagement.

Consider these factors:

**Positive Excitement Indicators:**
- Enthusiasm markers: ALL CAPS text, excessive exclamation marks (!!!, !!!!)
- "Instant classic" or "Game of the Year" references
- Phrases like "what a game", "incredible", "unbelievable", "amazing"
- Heart attack/stress references ("my heart can't take this", "cardiac arrest", "losing years off my life")
- Positive emotional outbursts and celebration
- High engagement (length of comments, detailed analysis)

**Engaged But Mixed Indicators:**
- Complaints about refs (shows engagement even if frustrated)
- Controversy discussion (indicates the game mattered)
- Stress and anxiety expressions (shows investment in outcome)

**Negative/Low Excitement Indicators:**
- Complaints about boring play
- "Not worth watching" or similar dismissive comments
- Apathy or disinterest
- Very short, low-effort comments
- Negative sentiment without engagement

**Scoring Guidelines:**
- 90-100: Overwhelming excitement, multiple "instant classic" references, high emotion
- 70-89: Strong positive sentiment, good engagement, fans clearly excited
- 50-69: Mixed sentiment, some excitement but also complaints
- 30-49: More negative than positive, low engagement
- 0-29: Apathy, dismissive comments, very low excitement

**Important Notes:**
- Weight recent comments higher than early thread comments (late game drama matters more)
- Distinguish between negative-but-engaged (refs controversy = still exciting) vs apathetic-negative (boring game)
- Look for consensus patterns across multiple comments
- Consider the emotional intensity, not just positive/negative polarity

IMPORTANT: Return as JSON with shape: { "score": 0, "sentiment": "excited|neutral|disappointed", "highlights": ["key theme 1", "key theme 2", "key theme 3"] }

Where:
- score: 0-100 excitement/engagement rating
- sentiment: overall fan sentiment category
- highlights: 2-4 key themes from the comments (e.g., "Instant classic references", "Ref complaints dominated discussion", "High stress/anxiety expressions")

Reddit comments:
`
