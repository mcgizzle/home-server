package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mcgizzle/home-server/apps/cloud/internal/application/consumers"
	usecases "github.com/mcgizzle/home-server/apps/cloud/internal/application/use_cases"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/database"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/queue"
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
	jobQueue queue.JobQueue,
) {
	processEvents := func() {
		log.Println("Checking for new events")
		competitions, err := fetchUseCase.Execute("nfl")
		if err != nil {
			log.Printf("Error fetching latest competitions: %v", err)
			return
		}

		err = saveUseCase.Execute(competitions)
		if err != nil {
			log.Printf("Error saving competitions: %v", err)
			return
		}
		log.Printf("Successfully processed %d competitions", len(competitions))

		// Generate any missing ratings after new competitions are saved
		if _, err := generateUseCase.Execute("nfl"); err != nil {
			log.Printf("Error generating ratings: %v", err)
		}

		// Schedule sentiment analysis jobs for completed games
		for _, comp := range competitions {
			if comp.Status == "final" || comp.Status == "completed" {
				// Schedule sentiment analysis for 30 minutes from now
				scheduledFor := time.Now().Add(30 * time.Minute)

				// Create job payload
				payload := map[string]string{
					"competition_id": comp.ID,
				}
				payloadJSON, err := json.Marshal(payload)
				if err != nil {
					log.Printf("Error marshaling sentiment job payload for competition %s: %v", comp.ID, err)
					continue
				}

				// Schedule the job
				job := queue.Job{
					ID:           uuid.New().String(),
					Type:         "sentiment_analysis",
					Payload:      payloadJSON,
					ScheduledFor: scheduledFor,
					CreatedAt:    time.Now(),
				}

				if err := jobQueue.Schedule(job); err != nil {
					log.Printf("Error scheduling sentiment job for competition %s: %v", comp.ID, err)
				} else {
					log.Printf("Scheduled sentiment analysis for competition %s at %s", comp.ID, scheduledFor.Format(time.RFC3339))
				}
			}
		}
	}

	// Run immediately on startup
	processEvents()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		processEvents()
	}
}

// backgroundFillMissingDetails periodically fetches and saves play-by-play for recent periods
func backgroundFillMissingDetails(
	fillUseCase usecases.FillMissingDetailsUseCase,
) {
	// Run immediately on startup
	log.Println("Filling missing details for recent periods")
	processed, updated, err := fillUseCase.Execute("nfl", 6)
	if err != nil {
		log.Printf("Detail fill run error: %v", err)
	} else {
		log.Printf("Detail fill run complete: processed=%d, updated=%d", processed, updated)
	}

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

	redditUserAgent := os.Getenv("REDDIT_USER_AGENT")
	if redditUserAgent == "" {
		redditUserAgent = "nfl-excitement-rating/1.0 by /u/nfl-app-user"
		log.Printf("REDDIT_USER_AGENT not set, using default: %s", redditUserAgent)
	}

	db := initDb()

	// Create dependencies
	espnClient := external.NewHTTPESPNClient()
	espnAdapter := external.NewESPNAdapter(espnClient)
	repo := sqliteinfra.NewSQLiteRepository(db)
	ratingService := external.NewOpenAIAdapter(openAIKey)

	// Sentiment analysis dependencies
	redditClient := external.NewHTTPRedditClient(redditUserAgent)
	sentimentService := external.NewSentimentAdapter(openAIKey)

	// Initialize job queue
	jobQueue := queue.NewSimpleQueue(100) // Buffer size of 100 jobs

	// Use cases
	getTemplateDataUseCase := usecases.NewGetTemplateDataUseCase(repo, repo)
	getAvailableDatesUseCase := usecases.NewGetAvailableDatesUseCase(repo)
	getLatestRatedDateUseCase := usecases.NewGetLatestRatedDateUseCase(repo)

	// Fetch/save and rating use cases
	fetchLatestUseCase := usecases.NewFetchLatestCompetitionsUseCase(espnAdapter, repo)
	saveUseCase := usecases.NewSaveCompetitionsUseCase(repo)
	generateRatingsUseCase := usecases.NewGenerateRatingsUseCase(repo, repo, ratingService)

	// Sentiment analysis use case
	generateSentimentUseCase := usecases.NewGenerateSentimentRatingUseCase(repo, repo, redditClient, sentimentService)

	// Start sentiment consumer
	ctx := context.Background()
	sentimentConsumer := consumers.NewSentimentConsumer(generateSentimentUseCase, jobQueue)
	go sentimentConsumer.Start(ctx)

	// Start background processes
	go func() {
		backgroundLatestEvents(fetchLatestUseCase, saveUseCase, generateRatingsUseCase, jobQueue)
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

	// Handler for rendering results page
	renderResults := func(w http.ResponseWriter, r *http.Request, season, week, periodType string) {
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
	}

	// Setup chi router
	r := chi.NewRouter()

	// Root path - show latest results
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		renderResults(w, r, "", "", "")
	})

	// Path-based URL: /season/{season}/week/{week}/type/{periodType}
	r.Get("/season/{season}/week/{week}/type/{periodType}", func(w http.ResponseWriter, r *http.Request) {
		season := chi.URLParam(r, "season")
		week := chi.URLParam(r, "week")
		periodType := chi.URLParam(r, "periodType")
		renderResults(w, r, season, week, periodType)
	})

	// Serve static files
	fileServer := http.FileServer(http.Dir("./static/"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8089"
	}
	log.Printf("Server started on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
