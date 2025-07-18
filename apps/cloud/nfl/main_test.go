package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// Mock implementations for testing
type MockESPNClient struct {
	eventsResponse external.LatestEvents
	specificEvents external.SpecificEvents
	event          external.EventResponse
	game           *domain.Game
}

func (m *MockESPNClient) ListLatestEvents() (external.LatestEvents, error) {
	return m.eventsResponse, nil
}

func (m *MockESPNClient) ListSpecificEvents(season, week, seasonType string) (external.SpecificEvents, error) {
	return m.specificEvents, nil
}

func (m *MockESPNClient) GetEvent(ref string) (external.EventResponse, error) {
	return m.event, nil
}

func (m *MockESPNClient) GetEventById(id string) (external.EventResponse, error) {
	return m.event, nil
}

func (m *MockESPNClient) GetScore(ref string) (external.ScoreResponse, error) {
	return external.ScoreResponse{}, nil
}

func (m *MockESPNClient) GetTeam(ref string) (external.TeamResponse, error) {
	return external.TeamResponse{}, nil
}

func (m *MockESPNClient) GetRecord(ref string) (external.RecordResponse, error) {
	return external.RecordResponse{}, nil
}

func (m *MockESPNClient) GetDetails(ref string, page int) (external.DetailsResponse, error) {
	return external.DetailsResponse{}, nil
}

func (m *MockESPNClient) GetDetailsPaged(ref string) ([]external.DetailsResponse, error) {
	return []external.DetailsResponse{}, nil
}

func (m *MockESPNClient) GetTeamAndScore(response external.EventResponse) *domain.Game {
	return m.game
}

type MockResultRepository struct {
	results []domain.Result
	dates   []domain.Date
	err     error
}

func (m *MockResultRepository) SaveResults(results []domain.Result) error {
	return m.err
}

func (m *MockResultRepository) LoadResults(season, week, seasonType string) ([]domain.Result, error) {
	return m.results, m.err
}

func (m *MockResultRepository) LoadDates() ([]domain.Date, error) {
	return m.dates, m.err
}

// Mock rating service that returns a fixed rating
type MockRatingService struct {
	rating domain.Rating
}

func (m *MockRatingService) ProduceRating(game domain.Game) domain.Rating {
	return m.rating
}

// Mock OpenAI server for testing
func startMockOpenAIServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock OpenAI response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": `{"score": 85, "explanation": "Great game with lots of excitement", "spoiler_free_explanation": "A thrilling matchup with big plays"}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestFetchLatestResultsUseCase(t *testing.T) {
	// Setup
	mockESPN := &MockESPNClient{
		eventsResponse: external.LatestEvents{
			Meta: struct {
				Parameters struct {
					Week        []string `json:"week"`
					Season      []string `json:"season"`
					SeasonTypes []string `json:"seasontypes"`
				} `json:"parameters"`
			}{
				Parameters: struct {
					Week        []string `json:"week"`
					Season      []string `json:"season"`
					SeasonTypes []string `json:"seasontypes"`
				}{
					Season:      []string{"2024"},
					Week:        []string{"1"},
					SeasonTypes: []string{"2"},
				},
			},
			Items: []external.EventRef{
				{Ref: "test-ref"},
			},
		},
		event: external.EventResponse{
			Id: "test-event-id",
		},
		game: &domain.Game{
			Home: domain.Team{Name: "Team A"},
			Away: domain.Team{Name: "Team B"},
		},
	}

	mockRepo := &MockResultRepository{
		results: []domain.Result{}, // No existing results
		err:     nil,
	}

	mockRatingSvc := &MockRatingService{
		rating: domain.Rating{
			Score:       85,
			Explanation: "Great game",
			SpoilerFree: "Thrilling matchup",
		},
	}

	useCase := application.NewFetchLatestResultsUseCase(mockESPN, mockRepo, mockRatingSvc)

	// Execute
	results, err := useCase.Execute()

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].EventId != "test-event-id" {
		t.Errorf("Expected event ID 'test-event-id', got %s", results[0].EventId)
	}
}

func TestFetchSpecificResultsUseCase(t *testing.T) {
	// Setup
	mockESPN := &MockESPNClient{
		specificEvents: external.SpecificEvents{
			Events: []external.EventId{
				{Id: "test-event-id"},
			},
		},
		event: external.EventResponse{
			Id: "test-event-id",
		},
		game: &domain.Game{
			Home: domain.Team{Name: "Team A"},
			Away: domain.Team{Name: "Team B"},
		},
	}

	mockRepo := &MockResultRepository{}
	mockRatingSvc := &MockRatingService{
		rating: domain.Rating{
			Score:       90,
			Explanation: "Excellent game",
			SpoilerFree: "Amazing matchup",
		},
	}

	useCase := application.NewFetchSpecificResultsUseCase(mockESPN, mockRepo, mockRatingSvc)

	// Execute
	results, err := useCase.Execute("2024", "1", "2")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].EventId != "test-event-id" {
		t.Errorf("Expected event ID 'test-event-id', got %s", results[0].EventId)
	}
}

func TestGetTemplateDataUseCase(t *testing.T) {
	// Setup
	mockRepo := &MockResultRepository{
		results: []domain.Result{
			{
				EventId: "test-event",
				Season:  "2024",
				Week:    "1",
				Rating: domain.Rating{
					Score:       85,
					Explanation: "Great game",
					SpoilerFree: "Thrilling matchup",
				},
			},
		},
		dates: []domain.Date{
			{
				Season:     "2024",
				Week:       "1",
				SeasonType: "2",
			},
		},
	}

	useCase := application.NewGetTemplateDataUseCase(mockRepo)

	// Execute
	data, err := useCase.Execute("2024", "1", "2")

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(data.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(data.Results))
	}

	if len(data.Dates) != 1 {
		t.Errorf("Expected 1 date, got %d", len(data.Dates))
	}
}

func TestSaveResultsUseCase(t *testing.T) {
	// Setup
	mockRepo := &MockResultRepository{
		err: nil,
	}

	useCase := application.NewSaveResultsUseCase(mockRepo)

	results := []domain.Result{
		{
			EventId: "test-event",
			Season:  "2024",
			Week:    "1",
			Rating: domain.Rating{
				Score:       85,
				Explanation: "Great game",
				SpoilerFree: "Thrilling matchup",
			},
		},
	}

	// Execute
	err := useCase.Execute(results)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Golden master test using real ESPN API data but mock OpenAI
func TestGoldenMaster(t *testing.T) {
	// Skip if not running golden master tests
	if os.Getenv("GOLDEN_MASTER") != "true" {
		t.Skip("Golden master test skipped - set GOLDEN_MASTER=true to run")
	}

	// Start mock OpenAI server
	server := startMockOpenAIServer()
	defer server.Close()

	// Set environment variable to use mock server
	os.Setenv("OPENAI_API_URL", server.URL)
	defer os.Unsetenv("OPENAI_API_URL")

	// Create real dependencies
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Initialize database
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create real dependencies
	resultRepo := repository.NewSQLiteResultRepository(db)
	espnClient := external.NewHTTPESPNClient()
	ratingSvc := application.NewOpenAIRatingService("test-key") // Key doesn't matter for mock server

	// Create use case
	fetchLatestUseCase := application.NewFetchLatestResultsUseCase(espnClient, resultRepo, ratingSvc)

	// Execute use case
	results, err := fetchLatestUseCase.Execute()

	// Basic assertions
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Log results for manual inspection
	log.Printf("Golden master test produced %d results", len(results))
	for i, result := range results {
		log.Printf("Result %d: EventId=%s, Season=%s, Week=%s, Score=%d",
			i+1, result.EventId, result.Season, result.Week, result.Rating.Score)
	}

	// Note: This test uses real ESPN API data but mock OpenAI responses
	// The exact results will depend on the current NFL season and available data
	// This is more of a smoke test to ensure the system works end-to-end
}
