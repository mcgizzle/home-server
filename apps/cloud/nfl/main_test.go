package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application"
	"github.com/mcgizzle/home-server/apps/cloud/internal/application/use_cases"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// Mock OpenAI server to avoid real API calls
func setupMockOpenAI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": `{"score": 75, "explanation": "This was a mock response for testing purposes. The game was quite entertaining with some notable plays that would generate a rant score of 7 out of 10.", "spoiler_free_explanation": "A competitive game with exciting moments and good quarterback play."}`,
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			http.NotFound(w, r)
		}
	}))
}

func TestRealESPNClient(t *testing.T) {
	// Create real ESPN client
	espnClient := external.NewHTTPESPNClient()

	// Test fetching latest events
	events, err := espnClient.ListLatestEvents()
	if err != nil {
		t.Fatalf("Failed to fetch latest events from real ESPN API: %v", err)
	}

	// Verify we got some data
	if len(events.Items) == 0 {
		t.Log("Warning: No events returned from ESPN API")
	} else {
		t.Logf("Successfully fetched %d events from ESPN API", len(events.Items))
	}

	// Test fetching specific events if we have any
	if len(events.Items) > 0 {
		eventRef := events.Items[0]
		event, err := espnClient.GetEvent(eventRef.Ref)
		if err != nil {
			t.Fatalf("Failed to fetch specific event from real ESPN API: %v", err)
		}

		t.Logf("Successfully fetched event %s from ESPN API", event.Id)
	}

	// Test fetching specific week/season
	specificEvents, err := espnClient.ListSpecificEvents("2024", "1", "2")
	if err != nil {
		t.Fatalf("Failed to fetch specific events from real ESPN API: %v", err)
	}

	t.Logf("Successfully fetched %d specific events from ESPN API", len(specificEvents.Events))
}

func TestRealESPNClientWithGameData(t *testing.T) {
	// Create real ESPN client
	espnClient := external.NewHTTPESPNClient()

	// Test fetching latest events and getting game data
	events, err := espnClient.ListLatestEvents()
	if err != nil {
		t.Fatalf("Failed to fetch latest events: %v", err)
	}

	if len(events.Items) == 0 {
		t.Skip("No events available to test game data")
	}

	// Get the first event
	eventRef := events.Items[0]
	event, err := espnClient.GetEvent(eventRef.Ref)
	if err != nil {
		t.Fatalf("Failed to fetch event: %v", err)
	}

	// Convert to domain game
	game := espnClient.GetTeamAndScore(event)
	if game == nil {
		t.Log("Game conversion returned nil (likely a live game, which is expected)")
		t.Skip("Skipping game data test - game is likely live or unavailable")
	}

	t.Logf("Successfully converted ESPN event to game: %s vs %s", game.Away.Name, game.Home.Name)
	t.Logf("Game has %d details", len(game.Details))
}

func TestRealESPNWithUseCases(t *testing.T) {
	// Setup mock OpenAI server
	mockOpenAI := setupMockOpenAI()
	defer mockOpenAI.Close()

	// Set environment variable to use mock OpenAI
	originalOpenAIURL := os.Getenv("OPENAI_API_URL")
	os.Setenv("OPENAI_API_URL", mockOpenAI.URL)
	defer os.Setenv("OPENAI_API_URL", originalOpenAIURL)

	// Create real dependencies
	espnClient := external.NewHTTPESPNClient()

	// Create in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatalf("Failed to create database schema: %v", err)
	}

	resultRepo := repository.NewSQLiteResultRepository(db)
	ratingSvc := application.NewOpenAIRatingService("test-key")

	// Create use cases
	fetchLatestUseCase := use_cases.NewFetchLatestResultsUseCase(espnClient, resultRepo, ratingSvc)
	saveUseCase := use_cases.NewSaveResultsUseCase(resultRepo)
	getTemplateDataUseCase := use_cases.NewGetTemplateDataUseCase(resultRepo)

	// Test fetching latest results
	results, err := fetchLatestUseCase.Execute()
	if err != nil {
		t.Fatalf("Failed to execute fetch latest use case: %v", err)
	}

	t.Logf("Successfully fetched %d results from real ESPN API", len(results))

	// Test saving results
	if len(results) > 0 {
		err = saveUseCase.Execute(results)
		if err != nil {
			t.Fatalf("Failed to save results: %v", err)
		}
		t.Logf("Successfully saved %d results to database", len(results))
	}

	// Test getting template data
	templateData, err := getTemplateDataUseCase.Execute("2024", "1", "2")
	if err != nil {
		t.Fatalf("Failed to get template data: %v", err)
	}

	t.Logf("Successfully retrieved template data with %d results", len(templateData.Results))
}

func TestRealESPNEndToEnd(t *testing.T) {
	// Setup mock OpenAI server
	mockOpenAI := setupMockOpenAI()
	defer mockOpenAI.Close()

	// Set environment variable to use mock OpenAI
	originalOpenAIURL := os.Getenv("OPENAI_API_URL")
	os.Setenv("OPENAI_API_URL", mockOpenAI.URL)
	defer os.Setenv("OPENAI_API_URL", originalOpenAIURL)

	// Create real dependencies
	espnClient := external.NewHTTPESPNClient()

	// Create in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// Initialize database schema
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatalf("Failed to create database schema: %v", err)
	}

	resultRepo := repository.NewSQLiteResultRepository(db)
	ratingSvc := application.NewOpenAIRatingService("test-key")

	// Create use cases
	fetchSpecificUseCase := use_cases.NewFetchSpecificResultsUseCase(espnClient, resultRepo, ratingSvc)
	saveUseCase := use_cases.NewSaveResultsUseCase(resultRepo)
	getTemplateDataUseCase := use_cases.NewGetTemplateDataUseCase(resultRepo)

	// Test fetching specific results (2024 season, week 1, regular season)
	results, err := fetchSpecificUseCase.Execute("2024", "1", "2")
	if err != nil {
		t.Fatalf("Failed to execute fetch specific use case: %v", err)
	}

	t.Logf("Successfully fetched %d specific results from real ESPN API", len(results))

	// Test saving results
	if len(results) > 0 {
		err = saveUseCase.Execute(results)
		if err != nil {
			t.Fatalf("Failed to save results: %v", err)
		}
		t.Logf("Successfully saved %d specific results to database", len(results))
	}

	// Test getting template data for the same period
	templateData, err := getTemplateDataUseCase.Execute("2024", "1", "2")
	if err != nil {
		t.Fatalf("Failed to get template data: %v", err)
	}

	t.Logf("Successfully retrieved template data with %d results", len(templateData.Results))
	t.Logf("Available dates: %d", len(templateData.Dates))

	// Test week display functionality
	if len(templateData.Dates) > 0 {
		for _, date := range templateData.Dates {
			t.Logf("Date display: %s - %s - %s", date.Season, date.WeekDisplay, date.SeasonTypeShowable)

			// Verify that post-season weeks are properly converted to round names
			if date.SeasonType == "3" { // Post-Season
				switch date.Week {
				case "1":
					if date.WeekDisplay != "Wild Card" {
						t.Errorf("Expected 'Wild Card' for post-season week 1, got '%s'", date.WeekDisplay)
					}
				case "2":
					if date.WeekDisplay != "Divisional" {
						t.Errorf("Expected 'Divisional' for post-season week 2, got '%s'", date.WeekDisplay)
					}
				case "3":
					if date.WeekDisplay != "Conference Championship" {
						t.Errorf("Expected 'Conference Championship' for post-season week 3, got '%s'", date.WeekDisplay)
					}
				case "4":
					if date.WeekDisplay != "Super Bowl" {
						t.Errorf("Expected 'Super Bowl' for post-season week 4, got '%s'", date.WeekDisplay)
					}
				}
			} else {
				// Verify that regular season weeks show "Week X"
				expected := "Week " + date.Week
				if date.WeekDisplay != expected {
					t.Errorf("Expected '%s' for season type %s week %s, got '%s'", expected, date.SeasonType, date.Week, date.WeekDisplay)
				}
			}
		}
	}
}
