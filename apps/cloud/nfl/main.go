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

	// Fetch/save and rating use cases
	fetchLatestUseCase := usecases.NewFetchLatestCompetitionsUseCase(espnAdapter, repo)
	saveUseCase := usecases.NewSaveCompetitionsUseCase(repo)
	generateRatingsUseCase := usecases.NewGenerateRatingsUseCase(repo, repo, ratingService)

	// Start background processes
	go func() {
		backgroundLatestEvents(fetchLatestUseCase, saveUseCase)
	}()
	go func() {
		generateRatingsUseCase.Execute("nfl")
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

		// Default parameter discovery
		if season == "" || week == "" || periodType == "" {
			dates, err := getAvailableDatesUseCase.Execute("nfl")
			if err != nil {
				log.Printf("Error loading dates: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if len(dates) > 0 {
				latestDate := dates[0]
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
