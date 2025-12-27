package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
)

func main() {
	var (
		eventID    = flag.String("event", "", "ESPN Event ID to export")
		search     = flag.String("search", "", "Search for games by team name (e.g., 'rams seahawks')")
		season     = flag.String("season", "2025", "Season year")
		week       = flag.String("week", "", "Week number (required for search)")
		seasonType = flag.String("type", "2", "Season type: 1=preseason, 2=regular, 3=playoff")
		outputDir  = flag.String("output", "eval/games", "Output directory for exported games")
		outputName = flag.String("name", "", "Custom output filename (without .json extension)")
		list       = flag.Bool("list", false, "List games for the specified season/week")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	client := external.NewHTTPESPNClient()

	switch {
	case *list:
		if *week == "" {
			log.Fatal("Week is required for listing games. Use -week=N")
		}
		listGames(client, *season, *week, *seasonType)

	case *search != "":
		if *week == "" {
			log.Fatal("Week is required for search. Use -week=N")
		}
		searchAndExport(client, *search, *season, *week, *seasonType, *outputDir, *outputName)

	case *eventID != "":
		exportGame(client, *eventID, *outputDir, *outputName)

	default:
		fmt.Println("Error: specify -event, -search, or -list")
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Eval Export - Export NFL games for promptfoo evaluation")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  eval-export -list -week=11                           # List games for week 11")
	fmt.Println("  eval-export -search 'rams seahawks' -week=11         # Find and export Rams vs Seahawks")
	fmt.Println("  eval-export -event=401772884                         # Export specific game by ID")
	fmt.Println("  eval-export -event=401772884 -name=rams_seahawks     # Export with custom filename")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("The exported JSON is compatible with promptfoo and includes:")
	fmt.Println("  - Home/away team names and scores")
	fmt.Println("  - Play-by-play data for AI analysis")
	fmt.Println("  - Game metadata (season, week, etc.)")
}

func listGames(client *external.HTTPESPNClient, season, week, seasonType string) {
	fmt.Printf("Fetching games for %s Season, Week %s...\n\n", season, week)

	events, err := client.ListSpecificEvents(season, week, seasonType)
	if err != nil {
		log.Fatalf("Failed to fetch events: %v", err)
	}

	fmt.Printf("%-12s %-45s %-10s\n", "Event ID", "Matchup", "Score")
	fmt.Println(strings.Repeat("-", 70))

	for _, eventRef := range events.Events {
		event, err := client.GetEventById(eventRef.Id)
		if err != nil {
			fmt.Printf("%-12s (failed to fetch)\n", eventRef.Id)
			continue
		}

		if len(event.Competitions) == 0 || len(event.Competitions[0].Competitors) < 2 {
			continue
		}

		comp := event.Competitions[0]
		var homeName, awayName string
		var homeScore, awayScore float64

		for _, competitor := range comp.Competitors {
			team, teamErr := client.GetTeam(competitor.Team.Ref)
			score, scoreErr := client.GetScore(competitor.Score.Ref)

			name := "Unknown"
			if teamErr == nil {
				name = team.DisplayName
			}
			scoreVal := 0.0
			if scoreErr == nil {
				scoreVal = score.Value
			}

			if competitor.HomeAway == "home" {
				homeName = name
				homeScore = scoreVal
			} else {
				awayName = name
				awayScore = scoreVal
			}
		}

		matchup := fmt.Sprintf("%s @ %s", awayName, homeName)
		scoreStr := fmt.Sprintf("%.0f-%.0f", awayScore, homeScore)

		fmt.Printf("%-12s %-45s %-10s\n", eventRef.Id, matchup, scoreStr)
	}

	fmt.Println()
	fmt.Println("Export a game with: eval-export -event=<ID>")
}

func searchAndExport(client *external.HTTPESPNClient, search, season, week, seasonType, outputDir, outputName string) {
	searchTerms := strings.Fields(strings.ToLower(search))
	fmt.Printf("Searching for games matching: %v\n", searchTerms)

	events, err := client.ListSpecificEvents(season, week, seasonType)
	if err != nil {
		log.Fatalf("Failed to fetch events: %v", err)
	}

	var matchedEventID string
	var matchedMatchup string

	for _, eventRef := range events.Events {
		event, err := client.GetEventById(eventRef.Id)
		if err != nil {
			continue
		}

		if len(event.Competitions) == 0 || len(event.Competitions[0].Competitors) < 2 {
			continue
		}

		comp := event.Competitions[0]
		var teamNames []string

		for _, competitor := range comp.Competitors {
			team, teamErr := client.GetTeam(competitor.Team.Ref)
			if teamErr == nil {
				teamNames = append(teamNames, strings.ToLower(team.DisplayName))
			}
		}

		// Check if all search terms match any team name
		allMatch := true
		for _, term := range searchTerms {
			found := false
			for _, name := range teamNames {
				if strings.Contains(name, term) {
					found = true
					break
				}
			}
			if !found {
				allMatch = false
				break
			}
		}

		if allMatch {
			matchedEventID = eventRef.Id
			matchedMatchup = strings.Join(teamNames, " vs ")
			break
		}
	}

	if matchedEventID == "" {
		log.Fatalf("No game found matching: %s", search)
	}

	fmt.Printf("Found: %s (ID: %s)\n", matchedMatchup, matchedEventID)
	exportGame(client, matchedEventID, outputDir, outputName)
}

func exportGame(client *external.HTTPESPNClient, eventID, outputDir, outputName string) {
	fmt.Printf("Fetching game data for event: %s\n", eventID)

	// Get event details
	event, err := client.GetEventById(eventID)
	if err != nil {
		log.Fatalf("Failed to fetch event: %v", err)
	}

	if len(event.Competitions) == 0 {
		log.Fatal("No competition data found")
	}

	comp := event.Competitions[0]

	// Build game structure
	game := make(map[string]interface{})

	for _, competitor := range comp.Competitors {
		team, teamErr := client.GetTeam(competitor.Team.Ref)
		score, scoreErr := client.GetScore(competitor.Score.Ref)

		teamName := "Unknown"
		if teamErr == nil {
			teamName = team.DisplayName
		}
		scoreVal := 0.0
		if scoreErr == nil {
			scoreVal = score.Value
		}

		teamData := map[string]interface{}{
			"name":  teamName,
			"score": scoreVal,
		}

		if competitor.HomeAway == "home" {
			game["home"] = teamData
		} else {
			game["away"] = teamData
		}
	}

	// Get play-by-play details
	fmt.Println("Fetching play-by-play data...")

	// Get details reference from competition
	detailsRef := comp.DetailsRefs.Ref
	if detailsRef == "" {
		fmt.Println("Warning: No play-by-play details available")
		game["details"] = []interface{}{}
	} else {
		detailsResponses, err := client.GetDetailsPaged(detailsRef)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch play-by-play: %v\n", err)
			game["details"] = []interface{}{}
		} else {
			var playByPlay []interface{}
			for _, detailsResponse := range detailsResponses {
				for _, item := range detailsResponse.Items {
					playByPlay = append(playByPlay, map[string]interface{}{
						"text": item.Text,
					})
				}
			}
			game["details"] = playByPlay
		}
	}

	// Determine output filename
	filename := outputName
	if filename == "" {
		// Generate from team names
		home := game["home"].(map[string]interface{})
		away := game["away"].(map[string]interface{})
		homeName := strings.ToLower(strings.ReplaceAll(home["name"].(string), " ", "_"))
		awayName := strings.ToLower(strings.ReplaceAll(away["name"].(string), " ", "_"))
		filename = fmt.Sprintf("%s_at_%s", awayName, homeName)
	}
	filename = strings.TrimSuffix(filename, ".json") + ".json"

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	outputPath := filepath.Join(outputDir, filename)

	// Write JSON file
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(game); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}

	fmt.Printf("\nExported to: %s\n", outputPath)

	// Print summary
	home := game["home"].(map[string]interface{})
	away := game["away"].(map[string]interface{})
	fmt.Printf("Game: %s @ %s\n", away["name"], home["name"])
	fmt.Printf("Score: %.0f - %.0f\n", away["score"], home["score"])
	if details, ok := game["details"].([]interface{}); ok {
		fmt.Printf("Play-by-play: %d plays\n", len(details))
	}

	fmt.Println("\nUse in promptfoo config:")
	fmt.Printf("  vars:\n    game: file://games/%s\n", filename)
}
