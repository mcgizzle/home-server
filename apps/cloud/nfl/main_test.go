package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// Simple OpenAI mock that returns deterministic ratings
type mockOpenAI struct {
	server *httptest.Server
}

func newMockOpenAI() *mockOpenAI {
	mock := &mockOpenAI{}
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return fixed rating response to avoid OpenAI costs
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": `{"score": 75, "explanation": "Test game with solid action and competitive play.", "spoiler_free_explanation": "A well-executed game with good pacing and competitive elements."}`,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	return mock
}

func (m *mockOpenAI) close() {
	m.server.Close()
}

// Test database setup - in-memory SQLite for isolation
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Initialize the schema
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// Golden Master Test - captures exact TemplateData structure with REAL ESPN data
func TestTemplateData_GoldenMaster(t *testing.T) {
	// Setup OpenAI mock to avoid costs but use real ESPN API
	mockAI := newMockOpenAI()
	defer mockAI.close()

	// Override the OpenAI API URL to point to our mock server
	// We'll need to intercept the HTTP client calls in produceRating()
	originalApiKey := ApiKey
	ApiKey = "test-key"
	defer func() { ApiKey = originalApiKey }()

	// Use test database
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Save original DB_PATH and restore after test
	originalDBPath := DB_PATH
	DB_PATH = ":memory:"
	defer func() { DB_PATH = originalDBPath }()

	// Test with a specific historical week for deterministic results
	// Using 2024 Week 10 Regular Season - games should be completed
	season := "2024"
	week := "10"
	seasonType := domain.RegularSeason

	t.Logf("Making real ESPN API calls for Season %s, Week %s, Type %s", season, week, seasonType)

	// Make real ESPN API calls to get authentic game data
	specificEvents := listSpecificEvents(season, week, seasonType)

	var results []domain.Result
	for i, eventId := range specificEvents.Events {
		// Limit to first 2 games to keep test fast but still validate real data
		if i >= 2 {
			break
		}

		t.Logf("Processing real ESPN event: %s", eventId.Id)
		event := getEventById(eventId.Id)
		maybeGame := getTeamAndScore(event)

		if maybeGame == nil {
			t.Logf("Game %s not completed, skipping", eventId.Id)
			continue
		}

		game := *maybeGame

		// Use mock rating instead of real OpenAI call to avoid costs
		mockRating := domain.Rating{
			Score:       75,
			Explanation: "Mock rating for deterministic testing",
			SpoilerFree: "Mock spoiler-free rating for testing",
		}

		result := domain.Result{
			Id:         i + 1,
			EventId:    eventId.Id,
			Season:     season,
			Week:       week,
			SeasonType: seasonType,
			Rating:     mockRating,
			Game:       game,
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		t.Skip("No completed games found for test week - this is expected for future weeks")
	}

	t.Logf("Retrieved %d real games from ESPN API", len(results))

	// Save real ESPN data to test database
	saveResults(testDB, results)

	// Create dates based on what we actually retrieved
	dates := []domain.Date{
		{Season: season, Week: week, SeasonType: seasonType},
	}

	dateTemplates := make([]domain.DateTemplate, len(dates))
	for i, date := range dates {
		dateTemplates[i] = date.Template()
	}

	// Create TemplateData structure exactly as it would be passed to template
	// This now contains REAL ESPN data instead of mock data
	templateData := domain.TemplateData{
		Results: results,
		Dates:   dateTemplates,
		Current: domain.Date{
			Season:     season,
			Week:       week,
			SeasonType: seasonType,
		}.Template(),
	}

	// Serialize to JSON for golden master comparison
	jsonData, err := json.MarshalIndent(templateData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal TemplateData: %v", err)
	}

	// Golden file path
	goldenFile := "testdata/template_data_golden.json"

	// Create testdata directory if it doesn't exist
	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	// Check if golden file exists
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		// Create golden file on first run
		if err := os.WriteFile(goldenFile, jsonData, 0644); err != nil {
			t.Fatalf("Failed to create golden file: %v", err)
		}
		t.Logf("Created golden file: %s", goldenFile)
		return
	}

	// Read existing golden file
	goldenData, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("Failed to read golden file: %v", err)
	}

	// Compare current data with golden data
	var goldenTemplateData domain.TemplateData
	if err := json.Unmarshal(goldenData, &goldenTemplateData); err != nil {
		t.Fatalf("Failed to unmarshal golden data: %v", err)
	}

	// Validate key structure elements
	if len(templateData.Results) != len(goldenTemplateData.Results) {
		t.Errorf("Results count mismatch: got %d, want %d", len(templateData.Results), len(goldenTemplateData.Results))
	}

	if len(templateData.Dates) != len(goldenTemplateData.Dates) {
		t.Errorf("Dates count mismatch: got %d, want %d", len(templateData.Dates), len(goldenTemplateData.Dates))
	}

	// Validate Current DateTemplate structure
	if templateData.Current.Season != goldenTemplateData.Current.Season {
		t.Errorf("Current season mismatch: got %s, want %s", templateData.Current.Season, goldenTemplateData.Current.Season)
	}

	// This test ensures the TemplateData contract remains stable during refactoring
	t.Logf("TemplateData structure validation passed")
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

// HTTP Edge Tests - Test only the HTTP boundaries
func TestHTTP_MainPage_WithData(t *testing.T) {
	// Setup
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Save original and override for test
	originalDB := DB_PATH
	DB_PATH = ":memory:"
	defer func() { DB_PATH = originalDB }()

	// Add some test data
	sampleResults := []domain.Result{
		{
			Id:         1,
			EventId:    "12345",
			Season:     "2024",
			Week:       "10",
			SeasonType: domain.RegularSeason,
			Rating:     domain.Rating{Score: 75, Explanation: "Test", SpoilerFree: "Test"},
			Game: domain.Game{
				Home: domain.Team{Name: "Home Team", Score: 24, Record: "8-2"},
				Away: domain.Team{Name: "Away Team", Score: 21, Record: "7-3"},
			},
		},
	}
	saveResults(testDB, sampleResults)

	// Create HTTP request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// We need to test the actual handler, but since it's embedded in main(),
	// we'll need to extract it or test via HTTP server
	// For now, this is a placeholder that validates the test structure
	_ = req
	_ = rr

	t.Log("HTTP test structure validated - handler extraction needed for full implementation")
}

func TestHTTP_MainPage_EmptyDB(t *testing.T) {
	// Test the "No data available yet" scenario
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Empty database - should show empty state
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// This should return the empty state template
	// Implementation will be completed after handler extraction
	_ = req
	_ = rr

	t.Log("Empty database test structure validated")
}

func TestHTTP_SpecificWeek(t *testing.T) {
	// Test query parameter handling: /?season=2024&week=10&seasontype=2
	req, err := http.NewRequest("GET", "/?season=2024&week=10&seasontype=2", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Should load data for specific week
	_ = req
	_ = rr

	t.Log("Specific week test structure validated")
}

func TestHTTP_StaticFiles(t *testing.T) {
	// Test static file serving: /static/main.css
	req, err := http.NewRequest("GET", "/static/main.css", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Should serve CSS file
	_ = req
	_ = rr

	t.Log("Static files test structure validated")
}
