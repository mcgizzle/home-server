package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	usecases "github.com/mcgizzle/home-server/apps/cloud/internal/application/use_cases"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/database"
	sqliteinfra "github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/sqlite"
)

var DB_PATH = "data/results.db"

func initDb() *sql.DB {
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal(err)
	}

	// Run migrations instead of manual table creation
	migrationsPath := "internal/infrastructure/migrations"
	err = database.RunMigrations(DB_PATH, migrationsPath)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migrations completed successfully")
	return db
}

// Background process using use cases
func backgroundLatestEvents(
	fetchUseCase usecases.FetchLatestCompetitionsUseCase,
	saveUseCase usecases.SaveCompetitionsUseCase,
	generateUseCase usecases.GenerateRatingsUseCase,
) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Checking for new events")
		competitions, err := fetchUseCase.Execute("nfl")
		if err != nil {
			log.Printf("Error fetching latest competitions: %v", err)
			continue
		}
		err = saveUseCase.Execute(competitions)
		if err != nil {
			log.Printf("Error saving competitions: %v", err)
			continue
		}
		log.Printf("Successfully processed %d competitions", len(competitions))

		// Generate any missing ratings after new competitions are saved
		if _, err := generateUseCase.Execute("nfl"); err != nil {
			log.Printf("Error generating ratings: %v", err)
		}
	}
}

// backgroundFillMissingDetails periodically fetches and saves play-by-play for recent periods
func backgroundFillMissingDetails(
	fillUseCase usecases.FillMissingDetailsUseCase,
) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Filling missing details for recent periods")
		processed, updated, err := fillUseCase.Execute("nfl", 6)
		if err != nil {
			log.Printf("Detail fill run error: %v", err)
			continue
		}
		log.Printf("Detail fill run complete: processed=%d, updated=%d", processed, updated)
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

	// Create dependencies
	espnClient := external.NewHTTPESPNClient()
	espnAdapter := external.NewESPNAdapter(espnClient)
	repo := sqliteinfra.NewSQLiteRepository(db)
	ratingService := external.NewOpenAIAdapter(openAIKey)

	// Use cases
	getTemplateDataUseCase := usecases.NewGetTemplateDataUseCase(repo)
	getAvailableDatesUseCase := usecases.NewGetAvailableDatesUseCase(repo)
	getLatestRatedDateUseCase := usecases.NewGetLatestRatedDateUseCase(repo)

	// Fetch/save and rating use cases
	fetchLatestUseCase := usecases.NewFetchLatestCompetitionsUseCase(espnAdapter, repo)
	saveUseCase := usecases.NewSaveCompetitionsUseCase(repo)
	generateRatingsUseCase := usecases.NewGenerateRatingsUseCase(repo, repo, ratingService)

	// Start background processes
	go func() {
		backgroundLatestEvents(fetchLatestUseCase, saveUseCase, generateRatingsUseCase)
	}()
	go func() {
		fillDetailsUseCase := usecases.NewFillMissingDetailsUseCase(espnAdapter, repo)
		backgroundFillMissingDetails(fillDetailsUseCase)
	}()

	// Load template
	tmpl, err := template.ParseFiles("static/template.html")
	if err != nil {
		log.Fatal(err)
	}

	// Main page handler using template data and available dates
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get query parameters with defaults
		season := r.URL.Query().Get("season")
		week := r.URL.Query().Get("week")
		periodType := r.URL.Query().Get("periodtype")

		// Default parameter discovery, prefer latest period that has any rating
		if season == "" || week == "" || periodType == "" {
			if date, ok, err := getLatestRatedDateUseCase.Execute("nfl"); err == nil && ok {
				if season == "" {
					season = date.Season
				}
				if week == "" {
					week = date.Period
				}
				if periodType == "" {
					periodType = date.PeriodType
				}
			} else {
				// Fallback: use most recent available date
				dates, err := getAvailableDatesUseCase.Execute("nfl")
				if err != nil {
					log.Printf("Error loading dates: %v", err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				if len(dates) > 0 {
					latestDate := dates[len(dates)-1]
					if season == "" {
						season = latestDate.Season
					}
					if week == "" {
						week = latestDate.Period
					}
					if periodType == "" {
						periodType = latestDate.PeriodType
					}
				}
			}
		}

		// Get template data
		templateData, err := getTemplateDataUseCase.Execute("nfl", season, week, periodType)
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
