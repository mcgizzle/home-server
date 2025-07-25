package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mcgizzle/home-server/apps/cloud/internal/application"
	"github.com/mcgizzle/home-server/apps/cloud/internal/application/use_cases"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

var DB_PATH = "data/results.db"

func initDb() *sql.DB {
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer unique, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}

	return db
}

func backgroundLatestEvents(ratingSvc application.RatingService, fetchLatestUseCase use_cases.FetchLatestResultsUseCase, saveUseCase use_cases.SaveResultsUseCase) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Checking for new events")
		newResults, err := fetchLatestUseCase.Execute()
		if err != nil {
			log.Printf("Error fetching latest results: %v", err)
			continue
		}
		err = saveUseCase.Execute(newResults)
		if err != nil {
			log.Printf("Error saving results: %v", err)
			continue
		}
	}
}

func main() {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	db := initDb()

	// Create dependencies
	resultRepo := repository.NewSQLiteResultRepository(db)
	espnClient := external.NewHTTPESPNClient()
	ratingSvc := application.NewOpenAIRatingService(openAIKey)

	// Create use cases
	fetchLatestUseCase := use_cases.NewFetchLatestResultsUseCase(espnClient, resultRepo, ratingSvc)
	fetchSpecificUseCase := use_cases.NewFetchSpecificResultsUseCase(espnClient, resultRepo, ratingSvc)
	saveUseCase := use_cases.NewSaveResultsUseCase(resultRepo)
	getTemplateDataUseCase := use_cases.NewGetTemplateDataUseCase(resultRepo)

	go func() {
		backgroundLatestEvents(ratingSvc, fetchLatestUseCase, saveUseCase)
	}()

	tmpl := template.Must(template.ParseFiles("static/template.html"))

	http.Handle("/run", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newResults, err := fetchLatestUseCase.Execute()
		if err != nil {
			log.Printf("Error fetching latest results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		err = saveUseCase.Execute(newResults)
		if err != nil {
			log.Printf("Error saving results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
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

		results, err := fetchSpecificUseCase.Execute(season, week, seasonType)
		if err != nil {
			log.Printf("Error fetching specific results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		err = saveUseCase.Execute(results)
		if err != nil {
			log.Printf("Error saving results: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		week := r.URL.Query().Get("week")
		season := r.URL.Query().Get("season")
		seasonType := r.URL.Query().Get("seasontype")

		// If any parameters are missing, find the best defaults
		if week == "" || season == "" || seasonType == "" {
			log.Printf("Missing query parameters, checking for existing data")

			// Check if any data exists at all
			dates, err := resultRepo.LoadDates()
			if err != nil {
				log.Printf("Error loading dates: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if len(dates) == 0 {
				log.Printf("No game data exists in database")
				http.Error(w, "No game data available. Please run /run to fetch the latest games or use /backfill to load specific weeks.", http.StatusNotFound)
				return
			}

			// Use the latest week with actual data
			latestDate := dates[0]
			if season == "" {
				season = latestDate.Season
			}
			if week == "" {
				week = latestDate.Week
			}
			if seasonType == "" {
				seasonType = latestDate.SeasonType
			}
			log.Printf("Using latest week with data: Season %s, Week %s, SeasonType %s", season, week, seasonType)
		}

		data, err := getTemplateDataUseCase.Execute(season, week, seasonType)
		if err != nil {
			log.Printf("Error getting template data: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Println(data.Seasons)

		err = tmpl.Execute(w, data)
		if err != nil {
			log.Printf("Error executing template: %v", err)
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
