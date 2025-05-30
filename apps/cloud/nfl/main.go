package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var ApiKey string
var DB_PATH = "data/results.db"

const PreSeason = "1"
const RegularSeason = "2"
const PostSeason = "3"

type EventRef struct {
	Ref string `json:"$ref"`
}

type LatestEvents struct {
	Items []EventRef `json:"items"`
	Meta  struct {
		Parameters struct {
			Week        []string `json:"week"`
			Season      []string `json:"season"`
			SeasonTypes []string `json:"seasontypes"`
		} `json:"parameters"`
	} `json:"$meta"`
}

// Fetches data for the current week's games
func listLatestEvents() LatestEvents {
	// make GET request to https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events

	req, err := http.NewRequest("GET", "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events", nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var res LatestEvents
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

type EventId struct {
	Id string `json:"id"`
}

type SpecificEvents struct {
	Events []EventId `json:"events"`
}

func listSpecificEvents(season string, week string, seasonType string) SpecificEvents {
	// https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?week=2&dates=2024&seasontype=3

	req, err := http.NewRequest("GET", fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?week=%s&dates=%s&seasontype=%s", week, season, seasonType), nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var res SpecificEvents
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

type EventResponse struct {
	Id           string         `json:"id"`
	Competitions []Competitions `json:"competitions"`
}

type Competitions struct {
	Competitors   []Competitors `json:"competitors"`
	DetailsRefs   DetailsRef    `json:"details"`
	LiveAvailable bool          `json:"liveAvailable"`
}

type Competitors struct {
	Id       string    `json:"id"`
	Team     TeamRef   `json:"team"`
	Score    ScoreRef  `json:"score"`
	HomeAway string    `json:"homeAway"`
	Record   RecordRef `json:"record"`
	Stats    StatsRef  `json:"statistics"`
}

type TeamRef struct {
	Ref string `json:"$ref"`
}

type ScoreRef struct {
	Ref string `json:"$ref"`
}

type RecordRef struct {
	Ref string `json:"$ref"`
}

func getEvent(ref string) EventResponse {

	req, err := http.NewRequest("GET", ref, nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Error closing response body: %s", err)
		}
	}(response.Body)

	var res EventResponse
	decoder := json.NewDecoder(response.Body)

	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

func getEventById(id string) EventResponse {

	req, err := http.NewRequest("GET", fmt.Sprintf("https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events/%s", id), nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var res EventResponse
	decoder := json.NewDecoder(response.Body)

	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

type ScoreResponse struct {
	Value float64 `json:"value"`
}

func getScore(ref string) ScoreResponse {

	req, err := http.NewRequest("GET", ref, nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(response.Body)

	var res ScoreResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

type TeamResponse struct {
	DisplayName string `json:"displayName"`
	Logos       []struct {
		Href string `json:"href"`
	} `json:"logos"`
}

func getTeam(ref string) TeamResponse {

	req, err := http.NewRequest("GET", ref, nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var res TeamResponse

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

type RecordResponse struct {
	Items []RecordItem `json:"items"`
}
type RecordItem struct {
	DisplayValue string `json:"displayValue"`
}

func getRecord(ref string) RecordResponse {

	req, err := http.NewRequest("GET", ref, nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var res RecordResponse

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		panic(err)
	}

	return res
}

type TeamResult struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Record string  `json:"record"`
	Logo   *string `json:"logo"`
}

type Game struct {
	Home    TeamResult    `json:"home"`
	Away    TeamResult    `json:"away"`
	Details []DetailsItem `json:"details"`
}

type StatsRef struct {
	Ref string `json:"$ref"`
}

type DetailsRef struct {
	Ref string `json:"$ref"`
}

type DetailsResponse struct {
	PageIndex int           `json:"pageIndex"`
	PageCount int           `json:"pageCount"`
	Items     []DetailsItem `json:"items"`
}

type DetailsItem struct {
	ShortText    string  `json:"shortText"`
	ScoringPlay  bool    `json:"scoringPlay"`
	ScoringValue float64 `json:"scoringValue"`
	Clock        struct {
		DisplayValue string `json:"displayValue"`
	}
}

func getDetails(ref string, page int) DetailsResponse {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s&page=%d", ref, page), nil)

	if err != nil {
		panic(err)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	var res DetailsResponse

	body, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &res)

	if err != nil {
		panic(err)
	}

	return res
}
func getDetailsPaged(ref string) []DetailsResponse {

	var details []DetailsResponse

	var res DetailsResponse
	res = getDetails(ref, 1)
	details = append(details, res)
	for i := 2; i <= res.PageCount; i++ {
		res := getDetails(ref, i)
		details = append(details, res)
	}
	return details
}

func getTeamAndScore(response EventResponse) *Game {
	competitors := response.Competitions[0].Competitors

	if response.Competitions[0].LiveAvailable {
		log.Printf("Game is live, skipping")
		return nil
	}

	var game Game

	for _, competitor := range competitors {

		team := getTeam(competitor.Team.Ref)
		score := getScore(competitor.Score.Ref)

		record := getRecord(competitor.Record.Ref)

		teamResult := TeamResult{
			Name:   team.DisplayName,
			Logo:   &team.Logos[0].Href,
			Score:  score.Value,
			Record: record.Items[0].DisplayValue,
		}

		if competitor.HomeAway == "home" {
			game.Home = teamResult
		} else {
			game.Away = teamResult
		}

	}
	// If game is not played yet, skip
	if game.Home.Score == 0 {
		return nil
	}

	details := response.Competitions[0].DetailsRefs

	var detailsItems []DetailsItem
	for _, detail := range getDetailsPaged(details.Ref) {
		detailsItems = append(detailsItems, detail.Items...)
	}

	game.Details = detailsItems

	return &game
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

type Rating struct {
	Score       int    `json:"score"`
	Explanation string `json:"explanation"`
	SpoilerFree string `json:"spoiler_free_explanation"`
}

func produceRating(game Game) Rating {
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

	post, err := client.R().SetAuthToken(ApiKey).SetBody(body).Post("https://api.openai.com/v1/chat/completions")
	if err != nil {
		return Rating{}
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

	var response Rating

	err = json.Unmarshal([]byte(jsonString), &response)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		os.Exit(2)

	}

	log.Printf("Response: %s", response.SpoilerFree)
	log.Printf("Rating Score: %d", response.Score)

	return response

}

type Result struct {
	Id         int    `json:"id"`
	EventId    string `json:"event_id"`
	Season     string `json:"season"`
	SeasonType string `json:"season_type"`
	Week       string `json:"week"`
	Rating     Rating `json:"rating"`
	Game       Game   `json:"game"`
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

func saveResults(db *sql.DB, results []Result) {

	stmt, err := db.Prepare("insert into results(event_id, season, week, season_type, rating, explanation, spoiler_free_explanation, game) values(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(stmt)

	for _, result := range results {

		gameJson, err := json.Marshal(result.Game)
		if err != nil {
			log.Fatal(err)
		}

		weekAsInt, err := strconv.Atoi(result.Week)
		if err != nil {
			log.Fatal(err)
		}
		seasonAsInt, err := strconv.Atoi(result.Season)
		if err != nil {
			log.Fatal(err)
		}

		seasonTypeAsInt, err := strconv.Atoi(result.SeasonType)
		if err != nil {
			log.Fatal(err)
		}

		eventIdAsInt, err := strconv.Atoi(result.EventId)
		if err != nil {
			log.Fatal(err)
		}

		_, err = stmt.Exec(eventIdAsInt, seasonAsInt, weekAsInt, seasonTypeAsInt, result.Rating.Score, result.Rating.Explanation, result.Rating.SpoilerFree, string(gameJson))
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("Saved %d results", len(results))
}

func loadResults(db *sql.DB, season string, week string, seasonType string) []Result {
	if season == "" || week == "" || seasonType == "" {
		log.Fatal("Season or week or season type not provided")
	}

	selectQuery := "select id, event_id, season, week, season_type, rating, explanation, spoiler_free_explanation, game from results where season = ? and week = ? and season_type = ? order by season desc, season_type desc, week desc, rating desc"

	rows, err := db.Query(selectQuery, season, week, seasonType)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	var results []Result

	for rows.Next() {
		var result Result
		var gameJson string
		err = rows.Scan(&result.Id, &result.EventId, &result.Season, &result.Week, &result.SeasonType, &result.Rating.Score, &result.Rating.Explanation, &result.Rating.SpoilerFree, &gameJson)

		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal([]byte(gameJson), &result.Game)
		if err != nil {
			log.Fatal(err)
		}

		results = append(results, result)
	}

	return results
}

func loadDates(db *sql.DB) []Date {
	selectQuery := "select distinct season, week, season_type from results order by season_type desc, season desc, week desc"

	rows, err := db.Query(selectQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	var dates []Date

	for rows.Next() {
		var season string
		var week string
		var seasonType string
		err = rows.Scan(&season, &week, &seasonType)

		if err != nil {
			log.Fatal(err)
		}

		dates = append(dates, Date{
			Season:     season,
			Week:       week,
			SeasonType: seasonType,
		})
	}

	return dates
}

func fetchResultsForThisWeek(existingResults []Result) []Result {

	eventRefs := listLatestEvents()

	season := eventRefs.Meta.Parameters.Season[0]
	week := eventRefs.Meta.Parameters.Week[0]
	seasonType := eventRefs.Meta.Parameters.SeasonTypes[0]

	// Filter out events that have already been processed
	var filteredEventRefs []EventRef
	for _, eventRef := range eventRefs.Items {
		event := getEvent(eventRef.Ref)
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

	var results []Result
	for _, eventRef := range filteredEventRefs {
		log.Printf("Processing event: Season %s - Week %s - Season Type %s", season, week, seasonType)
		event := getEvent(eventRef.Ref)
		maybeGame := getTeamAndScore(event)

		// Game has not been played yet
		if maybeGame == nil {
			log.Printf("Game has not been played yet, skipping")
			continue
		}
		game := *maybeGame

		rantScore := produceRating(game)

		result := Result{
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

func fetchResults(season string, week string, seasonType string) []Result {

	specificEvents := listSpecificEvents(season, week, seasonType)

	var results []Result
	for _, eventId := range specificEvents.Events {
		log.Printf("Processing event: %s - %s", season, week)
		event := getEventById(eventId.Id)
		maybeGame := getTeamAndScore(event)
		if maybeGame == nil {
			log.Printf("Game has not been played yet, skipping")
			continue
		}
		game := *maybeGame

		rantScore := produceRating(game)

		result := Result{
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

func backgroundLatestEvents(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Checking for new events")
			current := listLatestEvents().Meta.Parameters
			results := loadResults(db, current.Season[0], current.Week[0], current.SeasonTypes[0])
			newResults := fetchResultsForThisWeek(results)
			saveResults(db, newResults)

		}
	}
}

type Date struct {
	Season     string
	Week       string
	SeasonType string
}

// Displayed in the UI, seasontype is a string
type DateTemplate struct {
	Season     string
	Week       string
	SeasonType string
	// Printable version of season type
	SeasonTypeShowable string
}

func (d Date) Template() DateTemplate {

	var seasonType string
	switch d.SeasonType {
	case PreSeason:
		seasonType = "Preseason"
	case RegularSeason:
		seasonType = "Regular Season"
	case PostSeason:
		seasonType = "Postseason"
	}

	return DateTemplate{
		Season:             d.Season,
		Week:               d.Week,
		SeasonTypeShowable: seasonType,
		SeasonType:         d.SeasonType,
	}

}

func seasonTypeToNumber(seasonType string) string {
	switch seasonType {
	case PreSeason:
		return "1"
	case RegularSeason:
		return "2"
	case PostSeason:
		return "3"
	default:
		return "0"
	}
}

type TemplateData struct {
	Results []Result
	Dates   []DateTemplate
	Current DateTemplate
}

func main() {

	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	ApiKey = openAIKey

	db := initDb()

	go backgroundLatestEvents(db)

	tmpl := template.Must(template.ParseFiles("static/template.html"))

	http.Handle("/run", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		current := listLatestEvents().Meta.Parameters
		results := loadResults(db, current.Season[0], current.Week[0], current.SeasonTypes[0])
		newResults := fetchResultsForThisWeek(results)
		saveResults(db, newResults)
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

		results := fetchResults(season, week, seasonType)
		saveResults(db, results)

		w.WriteHeader(http.StatusOK)
	}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		week := r.URL.Query().Get("week")
		season := r.URL.Query().Get("season")
		seasonType := r.URL.Query().Get("seasontype")

		var results []Result
		if week != "" && season != "" && seasonType != "" {
			seasonTypeNumber := seasonTypeToNumber(seasonType)
			results = loadResults(db, season, week, seasonTypeNumber)
		} else {
			current := listLatestEvents().Meta.Parameters
			week = current.Week[0]
			season = current.Season[0]
			seasonType = current.SeasonTypes[0]
			results = loadResults(db, season, week, seasonType)
		}

		log.Printf("Loaded %d results for season [%s] and week [%s] and season type [%s]", len(results), season, week, seasonType)
		dates := loadDates(db)

		log.Printf("Loaded %d weeks", len(dates))

		dateTemplates := make([]DateTemplate, len(dates))
		for i, date := range dates {
			dateTemplates[i] = date.Template()
		}

		data := TemplateData{
			Results: results,
			Dates:   dateTemplates,
			Current: Date{
				Season:     season,
				Week:       week,
				SeasonType: seasonType,
			}.Template(),
		}

		err := tmpl.Execute(w, data)

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
