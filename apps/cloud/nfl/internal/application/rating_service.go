package application

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// RatingService defines the interface for rating operations
type RatingService interface {
	ProduceRating(game domain.Game) domain.Rating
}

// OpenAIRatingService implements RatingService using OpenAI API
type OpenAIRatingService struct {
	apiKey string
	client *resty.Client
}

// NewOpenAIRatingService creates a new OpenAI rating service
func NewOpenAIRatingService(apiKey string) RatingService {
	return &OpenAIRatingService{
		apiKey: apiKey,
		client: resty.New(),
	}
}

const prompt = "Analyze the provided NFL game play-by-play data and generate a 'rant score' between 0 and 100, acting as a HARSH judge of the game's excitement and intensity. Consider these factors:" +
	"Close score: Games decided by one score (8 points or less) are preferred." +
	"Controversial calls: Penalties that are questionable or have a major impact on the game." +
	"Big plays: Include passes of 50+ yards, runs of 30+ yards, and all turnovers." +
	"Momentum shifts: Defined as a team scoring 14 unanswered points or having 2+ consecutive turnovers." +
	"Blowouts: Games with a margin of victory of 17+ points will receive a significantly lower score." +
	"Excitement Factor: Give a high score to games with EITHER multiple lead changes OR a comeback where a team overcame a 14+ point deficit to win/tie or almost win." +
	"Give extra weight to:" +
	"High completion percentages from both quarterbacks." +
	"Total passing yards exceeding 600 yards." +
	"A combined total of 5+ touchdown passes." +
	"Limited penalties called (under 10 total), especially pre-snap penalties and offensive holding." +
	"Big plays and crucial conversions (3rd/4th downs) occurring in the 4th quarter or overtime, especially during a comeback." +
	"The data covers the entire game.  I favor games with good quarterback play and a high quality of play with few penalties and balanced offenses." +
	"Consider the overall 'wow' factor of the game. Were there memorable moments or plays that would be discussed for years to come? Be a tough critic - only truly exceptional games should score above 90!" +
	"IMPORTANT: Return as JSON with shape : { 'score' : 0, 'explanation' : 'Your explanation here, may include game spoilers.', 'spoiler_free_explanation' : 'Your spoiler-free explanation here, do not include any details about the outcome of the game' }"

// ProduceRating generates a rating for a game using OpenAI API
func (s *OpenAIRatingService) ProduceRating(game domain.Game) domain.Rating {
	type Body struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	gameAsJson, err := json.Marshal(game)
	if err != nil {
		log.Printf("Error marshaling game: %v", err)
		return domain.Rating{}
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
				Content: prompt + string(gameAsJson),
			},
		},
	}

	// Use environment variable for API URL in tests
	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/chat/completions"
	}

	post, err := s.client.R().SetAuthToken(s.apiKey).SetBody(body).Post(apiURL)
	if err != nil {
		log.Printf("Error calling OpenAI API: %v", err)
		return domain.Rating{}
	}

	type OuterResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	var outerJsonResponse OuterResponse
	err = json.Unmarshal([]byte(post.String()), &outerJsonResponse)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return domain.Rating{}
	}

	if len(outerJsonResponse.Choices) == 0 {
		log.Printf("No choices in OpenAI response")
		return domain.Rating{}
	}

	jsonString := outerJsonResponse.Choices[0].Message.Content
	jsonString = strings.TrimPrefix(jsonString, "```json\n")
	jsonString = strings.TrimSuffix(jsonString, "\n```")
	jsonString = strings.ReplaceAll(jsonString, "\\n", "")
	jsonString = strings.ReplaceAll(jsonString, "\\\"", "\"")

	var response domain.Rating

	err = json.Unmarshal([]byte(jsonString), &response)
	if err != nil {
		log.Printf("Error unmarshaling rating JSON: %v", err)
		return domain.Rating{}
	}

	log.Printf("Response: %s", response.SpoilerFree)
	log.Printf("Rating Score: %d", response.Score)

	return response
}
