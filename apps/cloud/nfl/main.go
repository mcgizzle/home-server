package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/database"
	v2usecases "github.com/mcgizzle/home-server/apps/cloud/internal/v2/application/use_cases"
	v2repository "github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"
)

var DB_PATH = "data/results.db"

func initDb() *sql.DB {
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal(err)
	}

	// Run V2 migrations instead of manual table creation
	migrationsPath := "internal/infrastructure/migrations"
	err = database.RunMigrations(DB_PATH, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migrations completed successfully")
	return db
}

// V2 background process using pure V2 use cases
func backgroundLatestEvents(
	v2FetchUseCase v2usecases.FetchLatestCompetitionsUseCase,
	v2SaveUseCase v2usecases.SaveCompetitionsUseCase,
) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Checking for new events (V2)")
		competitions, err := v2FetchUseCase.Execute("nfl")
		if err != nil {
			log.Printf("Error fetching latest competitions: %v", err)
			continue
		}
		err = v2SaveUseCase.Execute(competitions)
		if err != nil {
			log.Printf("Error saving competitions: %v", err)
			continue
		}
		log.Printf("Successfully processed %d competitions (V2)", len(competitions))
	}
}

func main() {
	log.Println("Starting NFL Excitement Rating Service")

	// Check required environment variables
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	db := initDb()

	// Create V2 dependencies - pure V2 system
	espnClient := external.NewHTTPESPNClient()
	espnAdapter := external.NewESPNAdapter(espnClient)
	v2Repo := v2repository.NewSQLiteV2Repository(db)
	v2RatingService := external.NewOpenAIAdapter(openAIKey)

	// V2 use cases replacing V1 equivalents
	v2GetTemplateDataUseCase := v2usecases.NewGetTemplateDataUseCase(v2Repo)
	v2GetAvailableDatesUseCase := v2usecases.NewGetAvailableDatesUseCase(v2Repo)

	// NEW: V2 fetch and save use cases for pure V2 pipeline
	v2FetchLatestUseCase := v2usecases.NewFetchLatestCompetitionsUseCase(espnAdapter, v2Repo)
	v2FetchSpecificUseCase := v2usecases.NewFetchSpecificCompetitionsUseCase(espnAdapter, v2Repo)
	v2SaveUseCase := v2usecases.NewSaveCompetitionsUseCase(v2Repo)
	v2GenerateRatingsUseCase := v2usecases.NewGenerateRatingsUseCase(v2Repo, v2Repo, v2RatingService)
	v2BackfillSeasonUseCase := v2usecases.NewBackfillSeasonUseCase(v2Repo, v2FetchSpecificUseCase, v2SaveUseCase)

	// Start background process using V2 - REPLACED V1 WITH V2
	go func() {
		backgroundLatestEvents(v2FetchLatestUseCase, v2SaveUseCase)
	}()
	go func() {
		v2GenerateRatingsUseCase.Execute("nfl")
	}()

	// Load template
	tmpl, err := template.ParseFiles("static/template.html")
	if err != nil {
		log.Fatal(err)
	}

	// REPLACED: /run endpoint now uses V2 pipeline
	http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		competitions, err := v2FetchLatestUseCase.Execute("nfl")
		if err != nil {
			log.Printf("Error fetching latest competitions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = v2SaveUseCase.Execute(competitions)
		if err != nil {
			log.Printf("Error saving competitions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully processed %d competitions via /run", len(competitions))
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/backfill", func(w http.ResponseWriter, r *http.Request) {
		week := r.URL.Query().Get("week")
		season := r.URL.Query().Get("season")
		periodType := r.URL.Query().Get("periodtype")

		if week == "" || season == "" || periodType == "" {
			http.Error(w, "Missing required parameters: week, season, periodtype", http.StatusBadRequest)
			return
		}

		// Use V2 specific fetch and save directly with semantic period type
		competitions, err := v2FetchSpecificUseCase.Execute("nfl", season, week, periodType)
		if err != nil {
			log.Printf("Error fetching specific competitions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = v2SaveUseCase.Execute(competitions)
		if err != nil {
			log.Printf("Error saving competitions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully processed %d competitions via /backfill", len(competitions))
		w.WriteHeader(http.StatusOK)
	})

	// NEW: Season backfill endpoint using V2 backfill use case
	http.HandleFunc("/backfill-season", func(w http.ResponseWriter, r *http.Request) {
		season := r.URL.Query().Get("season")
		if season == "" {
			http.Error(w, "Missing required parameter: season", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		var limit int
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "Invalid limit parameter: must be a number", http.StatusBadRequest)
				return
			}
			limit = parsedLimit
		}

		if limit > 0 {
			log.Printf("Starting season backfill for season %s with limit %d", season, limit)
		} else {
			log.Printf("Starting season backfill for season %s", season)
		}

		var result *v2usecases.BackfillResult
		var err error

		if limit > 0 {
			result, err = v2BackfillSeasonUseCase.ExecuteWithLimit("nfl", season, limit)
		} else {
			result, err = v2BackfillSeasonUseCase.Execute("nfl", season)
		}

		if err != nil {
			log.Printf("Error during season backfill: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("Season backfill completed: %d periods processed, %d competitions added, %d errors",
			result.PeriodsProcessed, result.CompetitionsAdded, len(result.Errors))

		// Return JSON response with backfill results
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Main page handler using V2 for template data and available dates
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get query parameters with defaults
		season := r.URL.Query().Get("season")
		week := r.URL.Query().Get("week")
		periodType := r.URL.Query().Get("periodtype")

		// Default parameter discovery using V2
		if season == "" || week == "" || periodType == "" {
			dates, err := v2GetAvailableDatesUseCase.Execute("nfl")
			if err != nil {
				log.Printf("Error loading dates: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if len(dates) > 0 {
				latestV2Date := dates[0]
				if season == "" {
					season = latestV2Date.Season
				}
				if week == "" {
					week = latestV2Date.Period
				}
				if periodType == "" {
					periodType = latestV2Date.PeriodType
				}
			}
		}

		// Get template data using V2
		templateData, err := v2GetTemplateDataUseCase.Execute("nfl", season, week, periodType)
		if err != nil {
			log.Printf("Error getting template data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, templateData)
		if err != nil {
			log.Printf("Error executing template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Println("Server started on http://localhost:8089")
	if err := http.ListenAndServe(":8089", nil); err != nil {
		log.Fatal(err)
	}
}
