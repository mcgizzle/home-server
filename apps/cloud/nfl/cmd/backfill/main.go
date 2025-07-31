package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/database"
	v2usecases "github.com/mcgizzle/home-server/apps/cloud/internal/v2/application/use_cases"
	v2repository "github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"
)

var DB_PATH = "data/results.db"

func main() {
	var (
		season     = flag.String("season", "", "Season to backfill (required)")
		sport      = flag.String("sport", "nfl", "Sport to backfill")
		limit      = flag.Int("limit", 0, "Limit number of competitions to process (0 = no limit)")
		jsonOutput = flag.Bool("json", false, "Output results as JSON")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	if *season == "" {
		fmt.Fprintf(os.Stderr, "Error: season parameter is required\n\n")
		printHelp()
		os.Exit(1)
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
	v2Repo := v2repository.NewSQLiteV2Repository(db)

	// Create use cases
	v2FetchSpecificUseCase := v2usecases.NewFetchSpecificCompetitionsUseCase(espnClient, v2Repo)
	v2SaveUseCase := v2usecases.NewSaveCompetitionsUseCase(v2Repo)
	v2BackfillSeasonUseCase := v2usecases.NewBackfillSeasonUseCase(v2Repo, v2FetchSpecificUseCase, v2SaveUseCase)

	// Execute backfill
	var result *v2usecases.BackfillResult

	if *limit > 0 {
		log.Printf("Using competition limit: %d", *limit)
		result, err = v2BackfillSeasonUseCase.ExecuteWithLimit(*sport, *season, *limit)
	} else {
		result, err = v2BackfillSeasonUseCase.Execute(*sport, *season)
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

	if len(result.Errors) > 0 {
		os.Exit(1) // Exit with error code if there were errors
	}
}

func printHelp() {
	fmt.Printf(`NFL Backfill Tool

This tool systematically fetches missing competition data for a complete season.

Usage:
  %s -season YYYY [-sport SPORT] [-limit N] [-json]

Examples:
  %s -season 2024                    # Backfill 2024 NFL season
  %s -season 2023 -json             # Backfill 2023 NFL season with JSON output
  %s -season 2024 -limit 5          # Backfill 2024 season, stop after 5 competitions
  %s -season 2024 -sport nfl        # Explicitly specify sport (default: nfl)

Options:
  -season string    Season to backfill (required)
  -sport string     Sport to backfill (default: nfl)
  -limit int        Limit number of competitions to process (0 = no limit)
  -json             Output detailed results as JSON
  -help             Show this help message

The tool will:
1. Check all regular season weeks (1-18) and playoff weeks (1-4)
2. For each period, verify what competitions already exist
3. Fetch missing competition data from ESPN
4. Save new competitions to the database
5. Report progress and results

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func printSummary(result *v2usecases.BackfillResult) {
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
