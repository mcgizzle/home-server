package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

var ApiKey string
var DB_PATH = "data/results.db"

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

func produceRating(game domain.Game) domain.Rating {
	client := resty.New()

	type Body struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	gameAsJson, err := json.Marshal(game)

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

	post, err := client.R().SetAuthToken(ApiKey).SetBody(body).Post(apiURL)
	if err != nil {
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
		fmt.Println("Error unmarshaling JSON:", err)
		os.Exit(2)

	}

	jsonString := outerJsonResponse.Choices[0].Message.Content
	jsonString = strings.TrimPrefix(jsonString, "```json\n")
	jsonString = strings.TrimSuffix(jsonString, "\n```")
	jsonString = strings.ReplaceAll(jsonString, "\\n", "")
	jsonString = strings.ReplaceAll(jsonString, "\\\"", "\"")

	var response domain.Rating

	err = json.Unmarshal([]byte(jsonString), &response)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		os.Exit(2)

	}

	log.Printf("Response: %s", response.SpoilerFree)
	log.Printf("Rating Score: %d", response.Score)

	return response

}

func initDb() *sql.DB {
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}

	return db
}

func fetchResultsForThisWeek(espnClient external.ESPNClient, resultRepo repository.ResultRepository, existingResults []domain.Result) []domain.Result {

	eventRefs, err := espnClient.ListLatestEvents()
	if err != nil {
		log.Printf("Error listing latest events: %v", err)
		return []domain.Result{}
	}

	season := eventRefs.Meta.Parameters.Season[0]
	week := eventRefs.Meta.Parameters.Week[0]
	seasonType := eventRefs.Meta.Parameters.SeasonTypes[0]

	// Filter out events that have already been processed
	var filteredEventRefs []external.EventRef
	for _, eventRef := range eventRefs.Items {
		event, err := espnClient.GetEvent(eventRef.Ref)
		if err != nil {
			log.Printf("Error getting event: %v", err)
			continue
		}
		shouldInclude := true
		for _, result := range existingResults {
			if result.EventId == event.Id {
				shouldInclude = false
				log.Printf("Event already processed: %s - %s", season, week)
				break
			}
		}
		if shouldInclude {
			filteredEventRefs = append(filteredEventRefs, eventRef)
		}
	}

	var results []domain.Result
	for _, eventRef := range filteredEventRefs {
		log.Printf("Processing event: Season %s - Week %s - Season Type %s", season, week, seasonType)
		event, err := espnClient.GetEvent(eventRef.Ref)
		if err != nil {
			log.Printf("Error getting event: %v", err)
			continue
		}
		maybeGame := espnClient.GetTeamAndScore(event)

		// Game has not been played yet
		if maybeGame == nil {
			log.Printf("Game has not been played yet, skipping")
			continue
		}
		game := *maybeGame

		rantScore := produceRating(game)

		result := domain.Result{
			EventId:    event.Id,
			Season:     season,
			SeasonType: seasonType,
			Week:       week,
			Rating:     rantScore,
			Game:       game,
		}
		results = append(results, result)

	}

	log.Printf("Produced %d results", len(results))

	return results

}

func fetchResults(espnClient external.ESPNClient, season string, week string, seasonType string) []domain.Result {

	specificEvents, err := espnClient.ListSpecificEvents(season, week, seasonType)
	if err != nil {
		log.Printf("Error listing specific events: %v", err)
		return []domain.Result{}
	}

	var results []domain.Result
	for _, eventId := range specificEvents.Events {
		log.Printf("Processing event: %s - %s", season, week)
		event, err := espnClient.GetEventById(eventId.Id)
		if err != nil {
			log.Printf("Error getting event by ID: %v", err)
			continue
		}
		maybeGame := espnClient.GetTeamAndScore(event)
		if maybeGame == nil {
			log.Printf("Game has not been played yet, skipping")
			continue
		}
		game := *maybeGame

		rantScore := produceRating(game)

		result := domain.Result{
			EventId:    eventId.Id,
			Season:     season,
			SeasonType: seasonType,
			Week:       week,
			Rating:     rantScore,
			Game:       game,
		}
		results = append(results, result)
	}

	log.Printf("Produced %d results", len(results))

	return results
}

func backgroundLatestEvents(db *sql.DB, espnClient external.ESPNClient, resultRepo repository.ResultRepository) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Checking for new events")
		current, err := espnClient.ListLatestEvents()
		if err != nil {
			log.Printf("Error listing latest events: %v", err)
			continue
		}
		results, err := resultRepo.LoadResults(current.Meta.Parameters.Season[0], current.Meta.Parameters.Week[0], current.Meta.Parameters.SeasonTypes[0])
		if err != nil {
			log.Printf("Error loading results: %v", err)
			continue
		}
		newResults := fetchResultsForThisWeek(espnClient, resultRepo, results)
		err = resultRepo.SaveResults(newResults)
		if err != nil {
			log.Printf("Error saving results: %v", err)
			continue
		}
	}
}

func main() {

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	ApiKey = openAIKey

	db := initDb()

	// Create dependencies
	resultRepo := repository.NewSQLiteResultRepository(db)
	espnClient := external.NewHTTPESPNClient()

	go backgroundLatestEvents(db, espnClient, resultRepo)

	tmpl := template.Must(template.ParseFiles("static/template.html"))

	http.Handle("/run", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		current, err := espnClient.ListLatestEvents()
		if err != nil {
			log.Printf("Error listing latest events: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		results, err := resultRepo.LoadResults(current.Meta.Parameters.Season[0], current.Meta.Parameters.Week[0], current.Meta.Parameters.SeasonTypes[0])
		if err != nil {
			log.Printf("Error loading results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		newResults := fetchResultsForThisWeek(espnClient, resultRepo, results)
		err = resultRepo.SaveResults(newResults)
		if err != nil {
			log.Printf("Error saving results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	http.Handle("/backfill", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		week := r.URL.Query().Get("week")
		season := r.URL.Query().Get("season")
		seasonType := r.URL.Query().Get("seasontype")

		if week == "" || season == "" || seasonType == "" {
			http.Error(w, "Missing week or season", http.StatusBadRequest)
			return
		}

		results := fetchResults(espnClient, season, week, seasonType)
		err := resultRepo.SaveResults(results)
		if err != nil {
			log.Printf("Error saving results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		week := r.URL.Query().Get("week")
		season := r.URL.Query().Get("season")
		seasonType := r.URL.Query().Get("seasontype")

		var results []domain.Result
		var err error
		if week != "" && season != "" && seasonType != "" {
			seasonTypeNumber := domain.SeasonTypeToNumber(seasonType)
			results, err = resultRepo.LoadResults(season, week, seasonTypeNumber)
			if err != nil {
				log.Printf("Error loading results: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		} else {
			// Instead of using current week from ESPN API, use the most recent week with results
			dates, err := resultRepo.LoadDates()
			if err != nil {
				log.Printf("Error loading dates: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if len(dates) > 0 {
				mostRecentDate := dates[0]
				week = mostRecentDate.Week
				season = mostRecentDate.Season
				seasonType = mostRecentDate.SeasonType
				results, err = resultRepo.LoadResults(season, week, seasonType)
				if err != nil {
					log.Printf("Error loading results: %v", err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
			} else {
				// Database is empty - show empty state
				log.Println("Database is empty, showing empty state")

				data := domain.TemplateData{
					Results: []domain.Result{},
					Dates:   []domain.DateTemplate{},
					Current: domain.DateTemplate{
						Season:             "No data",
						Week:               "available",
						SeasonTypeShowable: "yet",
						SeasonType:         "",
					},
				}

				err := tmpl.Execute(w, data)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}

		log.Printf("Loaded %d results for season [%s] and week [%s] and season type [%s]", len(results), season, week, seasonType)
		dates, err := resultRepo.LoadDates()
		if err != nil {
			log.Printf("Error loading dates: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Loaded %d weeks", len(dates))

		dateTemplates := make([]domain.DateTemplate, len(dates))
		for i, date := range dates {
			dateTemplates[i] = date.Template()
		}

		data := domain.TemplateData{
			Results: results,
			Dates:   dateTemplates,
			Current: domain.Date{
				Season:     season,
				Week:       week,
				SeasonType: seasonType,
			}.Template(),
		}

		err = tmpl.Execute(w, data)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/main.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		http.ServeFile(w, r, "static/main.css")
	})
	// serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("Starting server on :8089")

	err := http.ListenAndServe(":8089", nil)
	if err != nil {
		log.Fatal(err)
	}

}
