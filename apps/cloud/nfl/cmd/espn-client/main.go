package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
)

func main() {
	var (
		command    = flag.String("cmd", "", "Command to run: list-events, get-event, get-team, get-score, list-specific, site-scoreboard, site-summary, raw-event")
		eventID    = flag.String("event", "", "Event ID for get-event command")
		teamRef    = flag.String("team", "", "Team reference URL for get-team command")
		scoreRef   = flag.String("score", "", "Score reference URL for get-score command")
		season     = flag.String("season", "2024", "Season year for list-specific command")
		week       = flag.String("week", "1", "Week number for list-specific command")
		seasonType = flag.String("seasontype", "2", "Season type for list-specific command (1=preseason, 2=regular, 3=playoff)")
		pretty     = flag.Bool("pretty", true, "Pretty print JSON output")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if *command == "" {
		fmt.Println("Error: command is required")
		printHelp()
		os.Exit(1)
	}

	// Create ESPN client
	client := external.NewHTTPESPNClient()

	switch *command {
	case "list-events":
		listLatestEvents(client, *pretty)
	case "get-event":
		if *eventID == "" {
			fmt.Println("Error: event ID is required for get-event command")
			os.Exit(1)
		}
		getEvent(client, *eventID, *pretty)
	case "get-team":
		if *teamRef == "" {
			fmt.Println("Error: team reference URL is required for get-team command")
			os.Exit(1)
		}
		getTeam(client, *teamRef, *pretty)
	case "get-score":
		if *scoreRef == "" {
			fmt.Println("Error: score reference URL is required for get-score command")
			os.Exit(1)
		}
		getScore(client, *scoreRef, *pretty)
	case "list-specific":
		listSpecificEvents(client, *season, *week, *seasonType, *pretty)
	case "site-scoreboard":
		siteScoreboard(*season, *week, *seasonType, *pretty)
	case "site-summary":
		if *eventID == "" {
			fmt.Println("Error: event ID is required for site-summary command")
			os.Exit(1)
		}
		siteSummary(*eventID, *pretty)
	case "raw-event":
		if *eventID == "" {
			fmt.Println("Error: event ID is required for raw-event command")
			os.Exit(1)
		}
		rawEventCall(*eventID, *pretty)
	default:
		fmt.Printf("Error: unknown command '%s'\n", *command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("ESPN Client - Explore ESPN API data")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  espn-client -cmd=<command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list-events                     List latest events from ESPN (Core API)")
	fmt.Println("  get-event -event=<id>           Get detailed event by ID (Core API)")
	fmt.Println("  get-team -team=<url>            Get team details by reference URL")
	fmt.Println("  get-score -score=<url>          Get score details by reference URL")
	fmt.Println("  list-specific                   List specific events by season/week/type")
	fmt.Println("  site-scoreboard                 Get scoreboard with rich data (Site API)")
	fmt.Println("  site-summary -event=<id>        Get event summary with start times (Site API)")
	fmt.Println("  raw-event -event=<id>           Raw HTTP call to Core API event endpoint")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -season=<year>      Season year (default: 2024)")
	fmt.Println("  -week=<number>      Week number (default: 1)")
	fmt.Println("  -seasontype=<type>  Season type: 1=preseason, 2=regular, 3=playoff (default: 2)")
	fmt.Println("  -pretty=<bool>      Pretty print JSON (default: true)")
	fmt.Println("  -help               Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List latest events")
	fmt.Println("  espn-client -cmd=list-events")
	fmt.Println()
	fmt.Println("  # Get specific event details")
	fmt.Println("  espn-client -cmd=get-event -event=401671708")
	fmt.Println()
	fmt.Println("  # List events for specific week")
	fmt.Println("  espn-client -cmd=list-specific -season=2024 -week=1 -seasontype=2")
	fmt.Println()
	fmt.Println("  # Get team details (use team reference URL from event data)")
	fmt.Println("  espn-client -cmd=get-team -team='https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/seasons/2024/teams/12'")
	fmt.Println()
	fmt.Println("  # Get rich scoreboard data with start times (Site API)")
	fmt.Println("  espn-client -cmd=site-scoreboard -season=2024 -week=1 -seasontype=2")
	fmt.Println()
	fmt.Println("  # Get event summary with start time (Site API)")
	fmt.Println("  espn-client -cmd=site-summary -event=401671789")
	fmt.Println()
	fmt.Println("  # Raw HTTP call to explore Core API event structure")
	fmt.Println("  espn-client -cmd=raw-event -event=401671789")
}

func printJSON(data interface{}, pretty bool) {
	var output []byte
	var err error

	if pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}

	fmt.Println(string(output))
}

func listLatestEvents(client *external.HTTPESPNClient, pretty bool) {
	fmt.Println("🔍 Fetching latest events from ESPN...")
	events, err := client.ListLatestEvents()
	if err != nil {
		log.Fatalf("Error fetching latest events: %v", err)
	}

	fmt.Printf("✅ Found %d event references\n", len(events.Items))
	fmt.Println("📊 Available parameters:")
	fmt.Printf("  - Weeks: %s\n", strings.Join(events.Meta.Parameters.Week, ", "))
	fmt.Printf("  - Seasons: %s\n", strings.Join(events.Meta.Parameters.Season, ", "))
	fmt.Printf("  - Season Types: %s\n", strings.Join(events.Meta.Parameters.SeasonTypes, ", "))
	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(events, pretty)
}

func getEvent(client *external.HTTPESPNClient, eventID string, pretty bool) {
	fmt.Printf("🔍 Fetching event details for ID: %s\n", eventID)

	// Construct the event URL
	eventURL := fmt.Sprintf("https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events/%s", eventID)

	event, err := client.GetEvent(eventURL)
	if err != nil {
		log.Fatalf("Error fetching event: %v", err)
	}

	fmt.Printf("✅ Event ID: %s\n", event.Id)
	fmt.Printf("✅ Number of competitions: %d\n", len(event.Competitions))

	if len(event.Competitions) > 0 {
		comp := event.Competitions[0]
		fmt.Printf("✅ Number of competitors: %d\n", len(comp.Competitors))
		fmt.Printf("✅ Live available: %t\n", comp.LiveAvailable)

		// Show team references for easy exploration
		fmt.Println("🏈 Team References:")
		for i, competitor := range comp.Competitors {
			fmt.Printf("  Team %d (%s): %s\n", i+1, competitor.HomeAway, competitor.Team.Ref)
			fmt.Printf("  Score %d: %s\n", i+1, competitor.Score.Ref)
		}
	}

	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(event, pretty)
}

func getTeam(client *external.HTTPESPNClient, teamRef string, pretty bool) {
	fmt.Printf("🔍 Fetching team details from: %s\n", teamRef)

	team, err := client.GetTeam(teamRef)
	if err != nil {
		log.Fatalf("Error fetching team: %v", err)
	}

	fmt.Printf("✅ Team Name: %s\n", team.DisplayName)
	fmt.Printf("✅ Number of logos: %d\n", len(team.Logos))

	if len(team.Logos) > 0 {
		fmt.Printf("✅ Logo URL: %s\n", team.Logos[0].Href)
	}

	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(team, pretty)
}

func getScore(client *external.HTTPESPNClient, scoreRef string, pretty bool) {
	fmt.Printf("🔍 Fetching score details from: %s\n", scoreRef)

	score, err := client.GetScore(scoreRef)
	if err != nil {
		log.Fatalf("Error fetching score: %v", err)
	}

	fmt.Printf("✅ Score Value: %.1f\n", score.Value)

	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(score, pretty)
}

func listSpecificEvents(client *external.HTTPESPNClient, season, week, seasonType string, pretty bool) {
	fmt.Printf("🔍 Fetching specific events for Season: %s, Week: %s, Type: %s\n", season, week, seasonType)

	events, err := client.ListSpecificEvents(season, week, seasonType)
	if err != nil {
		log.Fatalf("Error fetching specific events: %v", err)
	}

	fmt.Printf("✅ Found %d events\n", len(events.Events))

	fmt.Println("🏈 Event IDs:")
	for i, event := range events.Events {
		fmt.Printf("  %d. %s\n", i+1, event.Id)
	}

	fmt.Println()
	fmt.Println("💡 To get details for any event, use:")
	for _, event := range events.Events {
		fmt.Printf("  espn-client -cmd=get-event -event=%s\n", event.Id)
	}

	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(events, pretty)
}

// siteScoreboard fetches scoreboard data from the Site API which includes start times
func siteScoreboard(season, week, seasonType string, pretty bool) {
	fmt.Printf("🔍 Fetching Site API scoreboard for Season: %s, Week: %s, Type: %s\n", season, week, seasonType)

	url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?week=%s&dates=%s&seasontype=%s", week, season, seasonType)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching scoreboard: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Extract some key information
	if events, ok := data["events"].([]interface{}); ok {
		fmt.Printf("✅ Found %d events with rich data\n", len(events))

		fmt.Println("🏈 Events with Start Times:")
		for i, event := range events {
			if eventMap, ok := event.(map[string]interface{}); ok {
				id := eventMap["id"]
				name := eventMap["name"]
				date := eventMap["date"]
				status := ""
				if statusMap, ok := eventMap["status"].(map[string]interface{}); ok {
					if statusType, ok := statusMap["type"].(map[string]interface{}); ok {
						status = fmt.Sprintf("%v", statusType["description"])
					}
				}

				fmt.Printf("  %d. ID: %v, Name: %v\n", i+1, id, name)
				fmt.Printf("      Date: %v, Status: %s\n", date, status)
			}
		}
	}

	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(data, pretty)
}

// siteSummary fetches event summary from the Site API which includes detailed game info
func siteSummary(eventID string, pretty bool) {
	fmt.Printf("🔍 Fetching Site API summary for event: %s\n", eventID)

	url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/football/nfl/summary?event=%s", eventID)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching summary: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Extract key game information
	if header, ok := data["header"].(map[string]interface{}); ok {
		if league, ok := header["league"].(map[string]interface{}); ok {
			fmt.Printf("✅ League: %v\n", league["name"])
		}

		if competition, ok := header["competition"].(map[string]interface{}); ok {
			fmt.Printf("✅ Date: %v\n", competition["date"])
			if venue, ok := competition["venue"].(map[string]interface{}); ok {
				fmt.Printf("✅ Venue: %v\n", venue["fullName"])
			}

			if competitors, ok := competition["competitors"].([]interface{}); ok {
				fmt.Println("🏈 Teams:")
				for _, comp := range competitors {
					if compMap, ok := comp.(map[string]interface{}); ok {
						if team, ok := compMap["team"].(map[string]interface{}); ok {
							fmt.Printf("  %s (%s): %v\n", team["displayName"], compMap["homeAway"], team["abbreviation"])
						}
					}
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("📋 Raw JSON Response:")
	printJSON(data, pretty)
}

// rawEventCall makes a raw HTTP call to the Core API event endpoint to explore structure
func rawEventCall(eventID string, pretty bool) {
	fmt.Printf("🔍 Making raw HTTP call to Core API event endpoint for ID: %s\n", eventID)

	url := fmt.Sprintf("https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events/%s", eventID)
	fmt.Printf("🌐 URL: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("📊 Response Status: %s\n", resp.Status)
	fmt.Printf("📊 Response Headers:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	// Parse JSON to check structure
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("⚠️  Error parsing JSON: %v\n", err)
		fmt.Printf("📋 Raw response body:\n%s\n", string(body))
		return
	}

	// Analyze the top-level fields to see what's available
	fmt.Printf("📋 Top-level fields in response:\n")
	for key, value := range data {
		valueType := fmt.Sprintf("%T", value)
		fmt.Printf("  - %s (%s)\n", key, valueType)

		// Check for potential date fields
		if key == "date" || key == "startTime" || key == "scheduledDate" || key == "dateTime" || key == "gameTime" {
			fmt.Printf("    🕒 POTENTIAL DATE FIELD: %v\n", value)
		}
	}

	// Look for date fields in nested structures
	fmt.Printf("\n🔍 Searching for date-related fields in nested structures...\n")
	searchForDateFields(data, "")

	fmt.Println()
	fmt.Println("📋 Full Raw JSON Response:")
	printJSON(data, pretty)
}

// searchForDateFields recursively searches for date-related fields in nested JSON
func searchForDateFields(data interface{}, path string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			currentPath := path
			if currentPath != "" {
				currentPath += "."
			}
			currentPath += key

			// Check if this key suggests a date field
			if isDateField(key) {
				fmt.Printf("  🕒 Found potential date field at %s: %v\n", currentPath, value)
			}

			// Recurse into nested structures (but limit depth to avoid too much output)
			if len(strings.Split(currentPath, ".")) < 4 {
				searchForDateFields(value, currentPath)
			}
		}
	case []interface{}:
		for i, item := range v {
			currentPath := fmt.Sprintf("%s[%d]", path, i)
			if len(strings.Split(currentPath, ".")) < 4 {
				searchForDateFields(item, currentPath)
			}
		}
	}
}

// isDateField checks if a field name suggests it might contain date/time information
func isDateField(fieldName string) bool {
	lowerField := strings.ToLower(fieldName)
	dateKeywords := []string{
		"date", "time", "scheduled", "start", "end", "created", "updated", "modified",
		"datetime", "timestamp", "when", "at", "on", "schedule", "gamedate", "gametime",
	}

	for _, keyword := range dateKeywords {
		if strings.Contains(lowerField, keyword) {
			return true
		}
	}
	return false
}
