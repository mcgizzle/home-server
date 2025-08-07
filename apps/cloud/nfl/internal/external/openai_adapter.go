package external

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
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

	var outerJsonResponse OuterResponse
	err = json.Unmarshal([]byte(post.String()), &outerJsonResponse)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return domain.Rating{}, err
	}

	if len(outerJsonResponse.Choices) == 0 {
		log.Printf("No choices in OpenAI response")
		return domain.Rating{}, err
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

// ExcitementPrompt is the refined prompt for generating excitement ratings
const ExcitementPrompt = `
You are an expert NFL game analyst. Your task is to analyze the provided play-by-play data and generate an 'Excitement Score' from 0 to 100.

Core Philosophy: Reward Drama, Forgive Imperfection
Your primary goal is to identify and reward exciting, dramatic football. All games have minor flaws. Do not let typical imperfections (e.g., a few penalties, a single untimely turnover) overshadow the brilliance of a true classic. A game with multiple lead changes, a comeback, and a down-to-the-wire finish should receive a top-tier score even if it wasn't perfectly clean. The positive "Legendary Factors" should vastly outweigh minor negative elements. Context is everything.

Follow this scoring rubric precisely:

1. Foundational Score (Margin + Game Quality)
Establish a starting point by considering both the final margin and the overall quality and flow of the game.

For Close Games (decided by 1-8 points or a tie):

High-Quality Close Game (e.g., strong offensive output, big plays, good QB duel, back-and-forth scoring): Start the score in the 70-85 range.

Sloppy/Poorly-Played Close Game (e.g., a "punt-fest" defined by offensive ineptitude and constant mistakes): Start the score in the 50-65 range.

For Moderately Close Games (decided by 9-16 points):

High-Quality Game (e.g., an entertaining offensive shootout where one team pulled away late): Start the score in the 55-70 range.

Average or Lopsided Game (e.g., one team was clearly better, scoring was front-loaded): Start the score in the 40-55 range.

For Blowouts (decided by 17+ points):

The score is fundamentally capped. Maximum possible score is 40. If the margin is 25+, the score should not exceed 25.

2. Major Positive Modifiers (The "Legendary" Factors)
These are the most important elements. ADD significant points for these, as they are the ingredients of a classic game.

Late-Game Drama: A game-winning or game-tying score in the final two minutes of the 4th quarter or in overtime.

Major Comeback: A team overcomes a deficit of 14+ points to win or force overtime.

Multiple Lead Changes: The lead changes hands 3 or more times, especially in the second half.

"Wow" Factor: A signature, unforgettable play (e.g., "Hail Mary," stunning catch, game-sealing defensive stand).

3. Specific Gameplay Bonuses
Also ADD points for high-level execution and exciting play:

Elite QB Duel (Huge Bonus): A high-level quarterback battle is a primary driver of excitement. Look for high completion percentages, a combined total of 600+ passing yards, OR a combined total of 5+ touchdown passes.

Big Plays: Multiple explosive plays (passes of 50+ yards, runs of 30+ yards) or impactful defensive touchdowns.

Clutch Conversions: Critical 3rd or 4th down conversions in high-leverage moments of the 4th quarter or OT.

4. Severe Negative Modifiers (Apply Sparingly)
Only subtract points if the game's quality was so poor it actively ruined the viewing experience. Do not penalize a great game for routine football mistakes.

Game-Ruining Sloppiness: The game was a chore to watch, defined by constant penalty flags (18+), repeated unforced errors (bad snaps, dropped INTs), and an overall lack of professional execution from both sides.

Final Score Definition:
A score of 90+ is reserved for legendary, all-time classic games that feature a high-quality foundation and multiple "Legendary Factors."

Explanation Style:
In your explanation, justify the score with a narrative analysis that reflects the Core Philosophy. Focus on what made the game exciting. Do not reveal the inner workings of the scoring rubric. Avoid mentioning foundational scores or specific point values. Your analysis should read like an expert's summary, not a calculation.

IMPORTANT: Return as JSON with the following shape:

{
  "score": 0,
  "explanation": "Your detailed explanation here. You may include game spoilers and justify the score based on the rubric provided.",
  "spoiler_free_explanation": "Your spoiler-free explanation here. Describe the general flow and quality of the game without revealing the winner, final score, or specific game-changing plays."
}
`
