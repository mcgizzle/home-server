package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	sqliteinfra "github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/sqlite"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Command line flags
	var (
		gameID     = flag.String("game", "", "Game ID to evaluate (required)")
		promptFile = flag.String("prompt", "", "Path to custom prompt file (optional, uses default if not provided)")
		outputDir  = flag.String("output", "evaluations", "Output directory for results")
		dbPath     = flag.String("db", "data/results.db", "Path to database file")
		apiKey     = flag.String("key", "", "OpenAI API key (or use OPENAI_API_KEY env var)")
		listGames  = flag.Bool("list", false, "List available games with their IDs")
		verbose    = flag.Bool("v", false, "Verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "OpenAI Game Evaluation Client\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -list                           # List available games\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -game <id>                      # Evaluate game with default prompt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -game <id> -prompt custom.txt   # Evaluate with custom prompt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Get API key from flag or environment
	openAIKey := *apiKey
	if openAIKey == "" {
		openAIKey = os.Getenv("OPENAI_API_KEY")
	}
	if openAIKey == "" && !*listGames {
		log.Fatal("OpenAI API key required. Use -key flag or set OPENAI_API_KEY environment variable")
	}

	// Open database
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	repo := sqliteinfra.NewSQLiteV2Repository(db)

	// Handle list games command
	if *listGames {
		if err := listAvailableGames(repo); err != nil {
			log.Fatalf("Failed to list games: %v", err)
		}
		return
	}

	// Validate required parameters
	if *gameID == "" {
		fmt.Fprintf(os.Stderr, "Error: -game flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Get game from database
	competition, err := repo.GetCompetitionByID(*gameID)
	if err != nil {
		log.Fatalf("Failed to get game %s: %v", *gameID, err)
	}

	if *verbose {
		fmt.Printf("Found game: %s vs %s\n",
			getTeamName(competition, "away"),
			getTeamName(competition, "home"))
	}

	// Load prompt
	prompt, promptName, err := loadPrompt(*promptFile)
	if err != nil {
		log.Fatalf("Failed to load prompt: %v", err)
	}

	if *verbose {
		fmt.Printf("Using prompt: %s\n", promptName)
		fmt.Printf("Prompt length: %d characters\n", len(prompt))
	}

	// Create custom OpenAI adapter with the custom prompt
	adapter := external.NewCustomOpenAIAdapter(openAIKey, prompt)

	// Generate rating
	fmt.Println("Generating rating...")
	rating, err := adapter.ProduceRatingForCompetition(*competition)
	if err != nil {
		log.Fatalf("Failed to generate rating: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Save results
	if err := saveResults(*outputDir, *gameID, promptName, rating, *competition); err != nil {
		log.Fatalf("Failed to save results: %v", err)
	}

	// Display results
	fmt.Printf("\n=== EVALUATION RESULTS ===\n")
	fmt.Printf("Game: %s vs %s\n",
		getTeamName(competition, "away"),
		getTeamName(competition, "home"))
	fmt.Printf("Score: %d/100\n", rating.Score)
	fmt.Printf("Prompt: %s\n", promptName)
	fmt.Printf("\nExplanation:\n%s\n", rating.Explanation)
	fmt.Printf("\nSpoiler-Free:\n%s\n", rating.SpoilerFree)

	outputFile := filepath.Join(*outputDir, fmt.Sprintf("%s_%s_%d.json",
		*gameID, promptName, time.Now().Unix()))
	fmt.Printf("\nResults saved to: %s\n", outputFile)
}

func listAvailableGames(repo repository.CompetitionRepository) error {
	// Alternative approach: Get recent periods and fetch competitions
	dates, err := repo.GetAvailablePeriods("nfl")
	if err != nil {
		return fmt.Errorf("failed to get available periods: %w", err)
	}

	var allCompetitions []domain.Competition
	// Get competitions from the last few periods
	maxPeriods := 10
	if len(dates) < maxPeriods {
		maxPeriods = len(dates)
	}

	for i := 0; i < maxPeriods; i++ {
		date := dates[i]
		competitions, err := repo.FindByPeriod(date.Season, date.Period, date.PeriodType, "nfl")
		if err != nil {
			continue // Skip periods with errors
		}
		allCompetitions = append(allCompetitions, competitions...)
		if len(allCompetitions) >= 50 {
			break
		}
	}

	competitions := allCompetitions

	fmt.Printf("Available Games (showing last %d):\n\n", len(competitions))
	fmt.Printf("%-15s %-30s %-10s %-15s %-10s\n", "ID", "Matchup", "Score", "Season/Week", "Rating")
	fmt.Printf("%s\n", strings.Repeat("-", 85))

	for _, comp := range competitions {
		awayTeam := getTeamName(&comp, "away")
		homeTeam := getTeamName(&comp, "home")
		awayScore := getTeamScore(&comp, "away")
		homeScore := getTeamScore(&comp, "home")

		matchup := fmt.Sprintf("%s @ %s", awayTeam, homeTeam)
		if len(matchup) > 28 {
			matchup = matchup[:25] + "..."
		}

		score := fmt.Sprintf("%0.0f-%0.0f", awayScore, homeScore)
		seasonWeek := fmt.Sprintf("%s W%s", comp.Season, comp.Period)

		ratingStr := "No rating"
		if comp.Rating != nil {
			ratingStr = fmt.Sprintf("%d/100", comp.Rating.Score)
		}

		fmt.Printf("%-15s %-30s %-10s %-15s %-10s\n",
			comp.ID, matchup, score, seasonWeek, ratingStr)
	}

	fmt.Printf("\nUse: %s -game <ID> to evaluate a specific game\n", os.Args[0])
	return nil
}

func loadPrompt(promptFile string) (string, string, error) {
	if promptFile == "" {
		// Use default prompt from OpenAI adapter
		return external.ExcitementPrompt, "default", nil
	}

	// Load custom prompt from file
	content, err := os.ReadFile(promptFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to read prompt file %s: %w", promptFile, err)
	}

	// Extract filename without extension for naming
	base := filepath.Base(promptFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	return string(content), name, nil
}

func saveResults(outputDir, gameID, promptName string, rating domain.Rating, competition domain.Competition) error {
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%s_%s_%d.json", gameID, promptName, timestamp)
	filepath := filepath.Join(outputDir, filename)

	// Create comprehensive results structure
	results := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"game_id":     gameID,
		"prompt_name": promptName,
		"game_info": map[string]interface{}{
			"away_team":   getTeamName(&competition, "away"),
			"home_team":   getTeamName(&competition, "home"),
			"away_score":  getTeamScore(&competition, "away"),
			"home_score":  getTeamScore(&competition, "home"),
			"season":      competition.Season,
			"week":        competition.Period,
			"period_type": competition.PeriodType,
		},
		"rating": map[string]interface{}{
			"score":                    rating.Score,
			"explanation":              rating.Explanation,
			"spoiler_free_explanation": rating.SpoilerFree,
			"source":                   rating.Source,
			"type":                     rating.Type,
			"generated_at":             rating.GeneratedAt.Format(time.RFC3339),
		},
	}

	// Write to file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

func getTeamName(comp *domain.Competition, homeAway string) string {
	for _, team := range comp.Teams {
		if team.HomeAway == homeAway {
			return team.Team.Name
		}
	}
	return "Unknown"
}

func getTeamScore(comp *domain.Competition, homeAway string) float64 {
	for _, team := range comp.Teams {
		if team.HomeAway == homeAway {
			return team.Score
		}
	}
	return 0
}
