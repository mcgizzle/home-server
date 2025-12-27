package external

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mcgizzle/home-server/apps/cloud/internal/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// OpenAIAdapter implements the RatingService interface using OpenAI API
// This adapter converts OpenAI-specific responses to domain entities
type OpenAIAdapter struct {
	apiKey string
	client *resty.Client
	prompt string // Custom prompt, uses default if empty
}

// NewOpenAIAdapter creates a new OpenAI adapter that implements RatingService
func NewOpenAIAdapter(apiKey string) services.RatingService {
	return &OpenAIAdapter{
		apiKey: apiKey,
		client: resty.New(),
		prompt: ExcitementPrompt, // Use default prompt
	}
}

// NewCustomOpenAIAdapter creates a new OpenAI adapter with a custom prompt
func NewCustomOpenAIAdapter(apiKey, customPrompt string) services.RatingService {
	return &OpenAIAdapter{
		apiKey: apiKey,
		client: resty.New(),
		prompt: customPrompt,
	}
}

// ProduceRatingForCompetition generates a rating for a competition using OpenAI API
func (a *OpenAIAdapter) ProduceRatingForCompetition(comp domain.Competition) (domain.Rating, error) {
	type Body struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	// Guard: require play-by-play to be present
	if comp.Details == nil || comp.Details.PlayByPlay == nil || len(comp.Details.PlayByPlay) == 0 {
		return domain.Rating{}, fmt.Errorf("missing play-by-play; refusing to rate competition %s", comp.ID)
	}

	// Convert competition to game structure for OpenAI
	gameForAPI := a.convertCompetitionToGameStructure(comp)
	gameAsJson, err := json.Marshal(gameForAPI)
	if err != nil {
		log.Printf("Error marshaling game: %v", err)
		return domain.Rating{}, err
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
				Content: a.prompt + string(gameAsJson),
			},
		},
	}

	// Use environment variable for API URL in tests
	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = "https://api.openai.com/v1/chat/completions"
	}

	log.Printf("=== OpenAI Request ===")
	log.Printf("URL: %s", apiURL)

	// Log game data summary instead of full payload
	log.Printf("Game Data Summary:")
	log.Printf("  Payload size: %d bytes", len(gameAsJson))

	// Extract and log game summary
	if home, exists := gameForAPI["home"]; exists {
		if homeMap, ok := home.(map[string]interface{}); ok {
			log.Printf("  Home: %v (Score: %v)", homeMap["name"], homeMap["score"])
		}
	}
	if away, exists := gameForAPI["away"]; exists {
		if awayMap, ok := away.(map[string]interface{}); ok {
			log.Printf("  Away: %v (Score: %v)", awayMap["name"], awayMap["score"])
		}
	}

	// Log play-by-play summary
	if details, exists := gameForAPI["details"]; exists {
		if detailsSlice, ok := details.([]interface{}); ok {
			log.Printf("  Play-by-play: %d plays", len(detailsSlice))
			if len(detailsSlice) > 0 {
				log.Printf("  First play: %v", detailsSlice[0])
				if len(detailsSlice) > 1 {
					log.Printf("  Last play: %v", detailsSlice[len(detailsSlice)-1])
				}
			}
		}
	}
	log.Printf("======================")

	post, err := a.client.R().SetAuthToken(a.apiKey).SetBody(body).Post(apiURL)
	if err != nil {
		log.Printf("Error calling OpenAI API: %v", err)
		return domain.Rating{}, err
	}

	// Log the response received from OpenAI
	log.Printf("=== OpenAI Response ===")
	log.Printf("Status Code: %d", post.StatusCode())
	log.Printf("Response Headers: %v", post.Header())
	log.Printf("Raw Response Body: %s", post.String())
	log.Printf("=======================")

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
		return domain.Rating{}, fmt.Errorf("OpenAI API error: %s", errorResponse.Error.Message)
	}

	var outerJsonResponse OuterResponse
	err = json.Unmarshal([]byte(post.String()), &outerJsonResponse)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return domain.Rating{}, err
	}

	if len(outerJsonResponse.Choices) == 0 {
		log.Printf("No choices in OpenAI response")
		return domain.Rating{}, fmt.Errorf("no choices in OpenAI response")
	}

	jsonString := outerJsonResponse.Choices[0].Message.Content
	jsonString = strings.TrimPrefix(jsonString, "```json\n")
	jsonString = strings.TrimSuffix(jsonString, "\n```")
	jsonString = strings.ReplaceAll(jsonString, "\\n", "")
	jsonString = strings.ReplaceAll(jsonString, "\\\"", "\"")

	// Parse into temporary structure matching OpenAI response
	var tempResponse struct {
		Score                  int    `json:"score"`
		Explanation            string `json:"explanation"`
		SpoilerFreeExplanation string `json:"spoiler_free_explanation"`
	}

	err = json.Unmarshal([]byte(jsonString), &tempResponse)
	if err != nil {
		log.Printf("Error unmarshaling rating JSON: %v", err)
		return domain.Rating{}, err
	}

	// Create domain Rating entity
	rating := domain.Rating{
		Type:        domain.RatingTypeExcitement,
		Score:       tempResponse.Score,
		Explanation: tempResponse.Explanation,
		SpoilerFree: tempResponse.SpoilerFreeExplanation,
		Source:      domain.RatingSourceOpenAI,
		GeneratedAt: time.Now(),
	}

	log.Printf("Response: %s", rating.SpoilerFree)
	log.Printf("Rating Score: %d", rating.Score)

	return rating, nil
}

// convertCompetitionToGameStructure converts Competition to game structure for OpenAI API
func (a *OpenAIAdapter) convertCompetitionToGameStructure(comp domain.Competition) map[string]interface{} {
	game := map[string]interface{}{}

	if len(comp.Teams) >= 2 {
		for _, team := range comp.Teams {
			if team.HomeAway == domain.HomeAwayHome {
				game["home"] = map[string]interface{}{
					"name":   team.Team.Name,
					"score":  team.Score,
					"record": a.getRecordFromStats(team.Stats),
				}
			} else {
				game["away"] = map[string]interface{}{
					"name":   team.Team.Name,
					"score":  team.Score,
					"record": a.getRecordFromStats(team.Stats),
				}
			}
		}
	}

	// Add details if available
	if comp.Details != nil && comp.Details.PlayByPlay != nil {
		game["details"] = comp.Details.PlayByPlay
	} else {
		game["details"] = []interface{}{}
	}

	return game
}

// getRecordFromStats extracts record from team stats
func (a *OpenAIAdapter) getRecordFromStats(stats map[string]interface{}) string {
	if record, exists := stats["record"]; exists {
		if recordStr, ok := record.(string); ok {
			return recordStr
		}
	}
	return ""
}

// ExcitementPrompt is the prompt for generating excitement ratings
const ExcitementPrompt = "Analyze the provided NFL game play-by-play data and generate a 'rant score' between 0 and 100, acting as a HARSH judge of the game's excitement and intensity. Consider these factors:" +
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
	"The data covers the entire game. I favor games with good quarterback play and a high quality of play with few penalties and balanced offenses." +
	"Consider the overall 'wow' factor of the game. Were there memorable moments or plays that would be discussed for years to come? Be a tough critic - only truly exceptional games should score above 90!" +
	"IMPORTANT: Return as JSON with shape: { 'score': 0, 'explanation': 'Your explanation here, may include game spoilers.', 'spoiler_free_explanation': 'Your spoiler-free explanation here, do not include any details about the outcome of the game' }"
