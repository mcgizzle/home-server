package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
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

	// V2 fetch and save use cases for web server
	v2FetchLatestUseCase := v2usecases.NewFetchLatestCompetitionsUseCase(espnAdapter, v2Repo)
	v2SaveUseCase := v2usecases.NewSaveCompetitionsUseCase(v2Repo)
	v2GenerateRatingsUseCase := v2usecases.NewGenerateRatingsUseCase(v2Repo, v2Repo, v2RatingService)

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
