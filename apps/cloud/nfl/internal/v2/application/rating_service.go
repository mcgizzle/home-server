package application

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
)

// V2RatingService defines the interface for V2 rating operations
type V2RatingService interface {
	ProduceRatingForCompetition(comp domain.Competition) (domain.Rating, error)
}

// V2OpenAIRatingService implements V2RatingService using OpenAI API
type V2OpenAIRatingService struct {
	apiKey string
	client *resty.Client
}

// NewV2RatingService creates a new V2 OpenAI rating service
func NewV2RatingService(apiKey string) V2RatingService {
	return &V2OpenAIRatingService{
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

const prompt2 = `
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

JSON

{
  "score": 0,
  "explanation": "Your detailed explanation here. You may include game spoilers and justify the score based on the rubric provided.",
  "spoiler_free_explanation": "Your spoiler-free explanation here. Describe the general flow and quality of the game without revealing the winner, final score, or specific game-changing plays."
}
`

// ProduceRatingForCompetition generates a rating for a V2 competition using OpenAI API
func (s *V2OpenAIRatingService) ProduceRatingForCompetition(comp domain.Competition) (domain.Rating, error) {
	type Body struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	// Convert V2 Competition to game structure for OpenAI (same as V1 structure)
	gameForAPI := convertCompetitionToGameStructure(comp)
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
				Content: prompt2 + string(gameAsJson),
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

	post, err := s.client.R().SetAuthToken(s.apiKey).SetBody(body).Post(apiURL)
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

	// Create V2 Rating entity
	v2Rating := domain.Rating{
		Type:        domain.RatingTypeExcitement,
		Score:       tempResponse.Score,
		Explanation: tempResponse.Explanation,
		SpoilerFree: tempResponse.SpoilerFreeExplanation,
		Source:      domain.RatingSourceOpenAI,
		GeneratedAt: time.Now(),
	}

	log.Printf("Response: %s", v2Rating.SpoilerFree)
	log.Printf("Rating Score: %d", v2Rating.Score)

	return v2Rating, nil
}

// convertCompetitionToGameStructure converts V2 Competition to game structure for OpenAI API
func convertCompetitionToGameStructure(comp domain.Competition) map[string]interface{} {
	game := map[string]interface{}{}

	if len(comp.Teams) >= 2 {
		for _, team := range comp.Teams {
			if team.HomeAway == domain.HomeAwayHome {
				game["home"] = map[string]interface{}{
					"name":   team.Team.Name,
					"score":  team.Score,
					"record": getRecordFromStats(team.Stats),
				}
			} else {
				game["away"] = map[string]interface{}{
					"name":   team.Team.Name,
					"score":  team.Score,
					"record": getRecordFromStats(team.Stats),
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
func getRecordFromStats(stats map[string]interface{}) string {
	if record, exists := stats["record"]; exists {
		if recordStr, ok := record.(string); ok {
			return recordStr
		}
	}
	return ""
}
