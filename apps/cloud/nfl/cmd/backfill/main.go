package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	usecases "github.com/mcgizzle/home-server/apps/cloud/internal/application/use_cases"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/database"
	sqliteinfra "github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/sqlite"
)

var DB_PATH = "data/results.db"

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "NFL Backfill Tool\n\n")
		fmt.Fprintf(os.Stderr, "This tool systematically fetches missing competition data for a complete season.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s -season 2024                              # Backfill 2024 NFL season (add missing competitions)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -season 2024 -ratings                     # Backfill + generate AI ratings\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -season 2024 -ratings -sentiment          # Backfill + ratings + Reddit sentiment\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -season 2024 -period 17 -periodtype regular -ratings -sentiment  # Week 17 only\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -season 2023 -json                        # Backfill 2023 NFL season with JSON output\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -season 2024 -update                      # Update existing 2024 competitions\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nThe tool will:\n")
		fmt.Fprintf(os.Stderr, "1. Check all regular season weeks (1-18) and playoff weeks (1-4)\n")
		fmt.Fprintf(os.Stderr, "2. For each period, verify what competitions already exist\n")
		fmt.Fprintf(os.Stderr, "3. In normal mode: Fetch missing competition data from ESPN\n")
		fmt.Fprintf(os.Stderr, "   In update mode: Re-fetch existing competitions to update them\n")
		fmt.Fprintf(os.Stderr, "4. Save competitions to the database (new or updated)\n")
		fmt.Fprintf(os.Stderr, "5. If -ratings: Generate AI ratings for games without ratings\n")
		fmt.Fprintf(os.Stderr, "6. If -sentiment: Analyze Reddit sentiment for completed games\n")
		fmt.Fprintf(os.Stderr, "7. Report progress and results\n")
	}
}

func main() {
	var (
		season        = flag.String("season", "", "Season to backfill (required)")
		sport         = flag.String("sport", "nfl", "Sport to backfill")
		limit         = flag.Int("limit", 0, "Limit number of competitions to process (0 = no limit)")
		jsonOutput    = flag.Bool("json", false, "Output results as JSON")
		updateMode    = flag.Bool("update", false, "Update existing competitions (useful for filling missing start times)")
		period        = flag.String("period", "", "Specific period/week to process (e.g., 1)")
		periodType    = flag.String("periodtype", "", "Specific period type (regular|playoff|preseason)")
		generateRatings   = flag.Bool("ratings", false, "Generate AI ratings for games without ratings (requires OPENAI_API_KEY)")
		generateSentiment = flag.Bool("sentiment", false, "Generate Reddit sentiment analysis for completed games (requires OPENAI_API_KEY)")
		help          = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *season == "" {
		fmt.Fprintf(os.Stderr, "Error: season parameter is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check API key if ratings or sentiment requested
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if (*generateRatings || *generateSentiment) && openAIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required for -ratings or -sentiment")
	}

	log.Printf("Starting backfill for %s season %s", *sport, *season)

	// Initialize database
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	migrationsPath := "internal/infrastructure/migrations"
	err = database.RunMigrations(DB_PATH, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create dependencies
	espnClient := external.NewHTTPESPNClient()
	espnAdapter := external.NewESPNAdapter(espnClient)
	repo := sqliteinfra.NewSQLiteRepository(db)

	// Create use cases
	fetchSpecificUseCase := usecases.NewFetchSpecificCompetitionsUseCase(espnAdapter, repo)
	saveUseCase := usecases.NewSaveCompetitionsUseCase(repo)
	backfillSeasonUseCase := usecases.NewBackfillSeasonUseCase(repo, fetchSpecificUseCase, saveUseCase)

	// Determine command to execute
	var cmd string
	if *updateMode {
		if *limit > 0 {
			cmd = "update-with-limit"
		} else {
			cmd = "update"
		}
	} else {
		if *limit > 0 {
			cmd = "backfill-with-limit"
		} else {
			cmd = "backfill"
		}
	}

	// Execute backfill
	var result *usecases.BackfillResult

	// If period-scoped flags are provided, run period-specific flows
	if *period != "" && *periodType != "" {
		switch cmd {
		case "update", "update-with-limit":
			log.Printf("Update mode (period): %s %s", *periodType, *period)
			result, err = backfillSeasonUseCase.ExecutePeriodUpdate(*sport, *season, *period, *periodType)
		case "backfill", "backfill-with-limit":
			log.Printf("Backfill mode (period): %s %s", *periodType, *period)
			result, err = backfillSeasonUseCase.ExecutePeriod(*sport, *season, *period, *periodType)
		default:
			log.Fatalf("Unknown command: %s", cmd)
		}
	} else {
		switch cmd {
		case "update":
			log.Printf("Update mode: updating existing competitions")
			result, err = backfillSeasonUseCase.ExecuteUpdate(*sport, *season)
		case "update-with-limit":
			log.Printf("Update mode with competition limit: %d", *limit)
			result, err = backfillSeasonUseCase.ExecuteUpdateWithLimit(*sport, *season, *limit)
		case "backfill":
			log.Printf("Backfill mode: adding missing competitions")
			result, err = backfillSeasonUseCase.Execute(*sport, *season)
		case "backfill-with-limit":
			log.Printf("Backfill mode with competition limit: %d", *limit)
			result, err = backfillSeasonUseCase.ExecuteWithLimit(*sport, *season, *limit)
		default:
			log.Fatalf("Unknown command: %s", cmd)
		}
	}

	if err != nil {
		log.Fatalf("Backfill failed: %v", err)
	}

	// Output results
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(result); encodeErr != nil {
			log.Fatalf("Failed to encode JSON: %v", encodeErr)
		}
	} else {
		printSummary(result)
	}

	// Generate ratings if requested
	if *generateRatings {
		log.Println("\n=== Generating AI Ratings ===")
		ratingService := external.NewOpenAIAdapter(openAIKey)
		ratingsGenerated := 0

		// Get competitions that need ratings
		competitions := getCompetitionsForPeriod(repo, *season, *period, *periodType, *sport)
		for _, comp := range competitions {
			// Skip if already has a rating
			if comp.Rating != nil && comp.Rating.Score > 0 {
				continue
			}
			// Skip if not completed
			if comp.Status != "final" && comp.Status != "completed" {
				continue
			}

			log.Printf("Generating rating for %s...", getMatchupName(&comp))
			rating, err := ratingService.ProduceRatingForCompetition(comp)
			if err != nil {
				log.Printf("  ERROR: %v", err)
				continue
			}

			// Save rating to database
			if err := repo.SaveRating(comp.ID, rating); err != nil {
				log.Printf("  ERROR saving: %v", err)
				continue
			}

			log.Printf("  Score: %d", rating.Score)
			ratingsGenerated++
		}
		log.Printf("Generated %d ratings", ratingsGenerated)
	}

	// Generate sentiment if requested
	if *generateSentiment {
		log.Println("\n=== Generating Reddit Sentiment ===")
		redditClient := external.NewHTTPRedditClient("nfl-backfill/1.0")
		sentimentService := external.NewSentimentAdapter(openAIKey)
		sentimentGenerated := 0

		// Get competitions that need sentiment
		competitions := getCompetitionsForPeriod(repo, *season, *period, *periodType, *sport)
		for _, comp := range competitions {
			// Skip if not completed
			if comp.Status != "final" && comp.Status != "completed" {
				continue
			}

			// Check if already has sentiment
			if existing, _ := repo.GetSentimentRating(comp.ID); existing != nil {
				continue
			}

			// Get team names
			team1, team2 := getTeamNames(&comp)
			if team1 == "" || team2 == "" {
				continue
			}

			// Determine game date
			gameDate := time.Now()
			if comp.StartTime != nil {
				gameDate = *comp.StartTime
			}

			log.Printf("Searching Reddit for %s vs %s...", team1, team2)

			// Search for post-game thread
			posts, err := redditClient.SearchPostGameThread(team1, team2, gameDate)
			if err != nil || len(posts) == 0 {
				log.Printf("  No thread found")
				continue
			}

			threadURL := fmt.Sprintf("https://reddit.com%s", posts[0].Permalink)
			log.Printf("  Found: %s", posts[0].Title)

			// Fetch comments
			comments, err := redditClient.GetThreadComments(threadURL)
			if err != nil || len(comments) == 0 {
				log.Printf("  ERROR fetching comments: %v", err)
				continue
			}

			// Extract comment bodies
			var bodies []string
			for _, c := range comments {
				if len(c.Body) > 10 && c.Body != "[deleted]" && c.Body != "[removed]" {
					bodies = append(bodies, c.Body)
				}
			}

			if len(bodies) == 0 {
				log.Printf("  No valid comments")
				continue
			}

			log.Printf("  Analyzing %d comments...", len(bodies))

			// Run sentiment analysis
			sentimentRating, err := sentimentService.AnalyzeSentiment("reddit", threadURL, bodies)
			if err != nil {
				log.Printf("  ERROR: %v", err)
				continue
			}

			// Save to database
			if err := repo.SaveSentimentRating(sentimentRating, comp.ID); err != nil {
				log.Printf("  ERROR saving: %v", err)
				continue
			}

			log.Printf("  Score: %d, Sentiment: %s", sentimentRating.Score, sentimentRating.Sentiment)
			sentimentGenerated++

			// Rate limit to avoid hitting Reddit too hard
			time.Sleep(2 * time.Second)
		}
		log.Printf("Generated %d sentiment ratings", sentimentGenerated)
	}

	if len(result.Errors) > 0 {
		os.Exit(1) // Exit with error code if there were errors
	}
}

// Helper functions for ratings/sentiment generation

func getCompetitionsForPeriod(repo *sqliteinfra.SQLiteRepository, season, period, periodType, sport string) []domain.Competition {
	if period != "" && periodType != "" {
		comps, _ := repo.FindByPeriod(season, period, periodType, domain.Sport(sport))
		return comps
	}

	// Get all periods for the season
	var allComps []domain.Competition
	dates, _ := repo.GetAvailablePeriods(domain.Sport(sport))
	for _, d := range dates {
		if d.Season == season {
			comps, _ := repo.FindByPeriod(d.Season, d.Period, d.PeriodType, domain.Sport(sport))
			allComps = append(allComps, comps...)
		}
	}
	return allComps
}

func getMatchupName(comp *domain.Competition) string {
	var away, home string
	for _, t := range comp.Teams {
		if t.HomeAway == "away" {
			away = t.Team.Name
		} else if t.HomeAway == "home" {
			home = t.Team.Name
		}
	}
	return fmt.Sprintf("%s @ %s", away, home)
}

func getTeamNames(comp *domain.Competition) (string, string) {
	var team1, team2 string
	for _, t := range comp.Teams {
		name := t.Team.Name
		// Extract just the team name (last word) for Reddit search
		parts := strings.Fields(name)
		if len(parts) > 0 {
			shortName := parts[len(parts)-1]
			if t.HomeAway == "away" {
				team1 = shortName
			} else {
				team2 = shortName
			}
		}
	}
	return team1, team2
}

func printSummary(result *usecases.BackfillResult) {
	fmt.Printf("=== Backfill Summary for Season %s ===\n", result.Season)
	if result.Limit > 0 {
		fmt.Printf("Competition Limit: %d", result.Limit)
		if result.LimitReached {
			fmt.Printf(" (REACHED)")
		}
		fmt.Println()
	}
	fmt.Printf("Periods Processed: %d\n", result.PeriodsProcessed)
	fmt.Printf("Total Competitions Added: %d\n", result.CompetitionsAdded)
	fmt.Printf("Errors: %d\n", len(result.Errors))
	fmt.Println()

	if len(result.Errors) > 0 {
		fmt.Println("=== Errors ===")
		for _, err := range result.Errors {
			fmt.Printf("  %s %s: %s\n", err.Period, err.PeriodType, err.Error)
		}
		fmt.Println()
	}

	fmt.Println("=== Period Details ===")
	regularCount, playoffCount := 0, 0
	regularAdded, playoffAdded := 0, 0

	for _, periodResult := range result.PeriodResults {
		if periodResult.PeriodType == "regular" {
			regularCount++
			regularAdded += periodResult.AddedCount
		} else if periodResult.PeriodType == "playoff" {
			playoffCount++
			playoffAdded += periodResult.AddedCount
		}

		status := "✓"
		if periodResult.Error != "" {
			status = "✗"
		} else if periodResult.Skipped {
			status = "⊘"
		}

		fmt.Printf("  %s Week %s: %s %d existing, %d added",
			periodResult.PeriodType, periodResult.Period, status,
			periodResult.ExistingCount, periodResult.AddedCount)

		if periodResult.SkipReason != "" {
			fmt.Printf(" (%s)", periodResult.SkipReason)
		}
		if periodResult.Error != "" {
			fmt.Printf(" - ERROR: %s", periodResult.Error)
		}
		fmt.Println()
	}

	fmt.Printf("\nRegular Season: %d periods, %d competitions added\n", regularCount, regularAdded)
	fmt.Printf("Playoffs: %d periods, %d competitions added\n", playoffCount, playoffAdded)
}
