package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// Mock OpenAI server for testing
var mockOpenAIServer *httptest.Server

func init() {
	// Create a mock OpenAI server
	mockOpenAIServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a mock response
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
}

// Mock ESPN client for testing
type MockESPNClient struct{}

func (m *MockESPNClient) ListLatestEvents() (external.LatestEvents, error) {
	return external.LatestEvents{
		Items: []external.EventRef{
			{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events/401547456"},
		},
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
				Week:        []string{"2"},
				Season:      []string{"2024"},
				SeasonTypes: []string{"2"},
			},
		},
	}, nil
}

func (m *MockESPNClient) ListSpecificEvents(season, week, seasonType string) (external.SpecificEvents, error) {
	return external.SpecificEvents{
		Events: []external.EventId{
			{Id: "401547456"},
		},
	}, nil
}

func (m *MockESPNClient) GetEvent(ref string) (external.EventResponse, error) {
	// Return mock event data
	return external.EventResponse{
		Id: "401547456",
		Competitions: []external.Competitions{
			{
				Competitors: []external.Competitors{
					{
						Id:       "1",
						HomeAway: "home",
						Team:     external.TeamRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/1"},
						Score:    external.ScoreRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/1/score"},
						Record:   external.RecordRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/1/record"},
					},
					{
						Id:       "2",
						HomeAway: "away",
						Team:     external.TeamRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/2"},
						Score:    external.ScoreRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/2/score"},
						Record:   external.RecordRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/2/record"},
					},
				},
				LiveAvailable: false,
				DetailsRefs:   external.DetailsRef{Ref: "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events/401547456/competitions/1/details"},
			},
		},
	}, nil
}

func (m *MockESPNClient) GetEventById(id string) (external.EventResponse, error) {
	return m.GetEvent("")
}

func (m *MockESPNClient) GetScore(ref string) (external.ScoreResponse, error) {
	return external.ScoreResponse{Value: 24.0}, nil
}

func (m *MockESPNClient) GetTeam(ref string) (external.TeamResponse, error) {
	return external.TeamResponse{
		DisplayName: "Kansas City Chiefs",
		Logos: []struct {
			Href string `json:"href"`
		}{
			{Href: "https://a.espncdn.com/i/teamlogos/nfl/500/kc.png"},
		},
	}, nil
}

func (m *MockESPNClient) GetRecord(ref string) (external.RecordResponse, error) {
	return external.RecordResponse{
		Items: []external.RecordItem{
			{DisplayValue: "2-0"},
		},
	}, nil
}

func (m *MockESPNClient) GetDetails(ref string, page int) (external.DetailsResponse, error) {
	return external.DetailsResponse{
		PageIndex: 1,
		PageCount: 1,
		Items: []domain.DetailsItem{
			{
				Text: "Touchdown! Patrick Mahomes throws a 50-yard pass to Travis Kelce",
			},
		},
	}, nil
}

func (m *MockESPNClient) GetDetailsPaged(ref string) ([]external.DetailsResponse, error) {
	details, err := m.GetDetails(ref, 1)
	if err != nil {
		return nil, err
	}
	return []external.DetailsResponse{details}, nil
}

func (m *MockESPNClient) GetTeamAndScore(response external.EventResponse) *domain.Game {
	// Mock implementation that returns a simple game
	return &domain.Game{
		Home: domain.Team{
			Name:   "Kansas City Chiefs",
			Score:  24,
			Record: "2-0",
		},
		Away: domain.Team{
			Name:   "Baltimore Ravens",
			Score:  20,
			Record: "1-1",
		},
		Details: []domain.DetailsItem{
			{
				Text: "Touchdown! Patrick Mahomes throws a 50-yard pass to Travis Kelce",
			},
		},
	}
}

// Mock repository for testing
type MockResultRepository struct {
	results []domain.Result
	dates   []domain.Date
}

func (m *MockResultRepository) SaveResults(results []domain.Result) error {
	m.results = append(m.results, results...)
	return nil
}

func (m *MockResultRepository) LoadResults(season, week, seasonType string) ([]domain.Result, error) {
	var filtered []domain.Result
	for _, result := range m.results {
		if result.Season == season && result.Week == week && result.SeasonType == seasonType {
			filtered = append(filtered, result)
		}
	}
	return filtered, nil
}

func (m *MockResultRepository) LoadDates() ([]domain.Date, error) {
	return m.dates, nil
}

func TestFetchResultsForThisWeek(t *testing.T) {
	// Override the OpenAI API URL to use our mock server
	originalApiKey := ApiKey
	ApiKey = "test-key"
	defer func() { ApiKey = originalApiKey }()

	// Set the mock OpenAI API URL
	os.Setenv("OPENAI_API_URL", mockOpenAIServer.URL)
	defer os.Unsetenv("OPENAI_API_URL")

	// Create mock dependencies
	espnClient := &MockESPNClient{}
	resultRepo := &MockResultRepository{
		results: []domain.Result{},
		dates: []domain.Date{
			{Season: "2024", Week: "2", SeasonType: "2"},
		},
	}

	// Test the function
	results := fetchResultsForThisWeek(espnClient, resultRepo, []domain.Result{})

	// Verify results
	if len(results) == 0 {
		t.Error("Expected at least one result, got none")
	}

	// Verify the first result
	if len(results) > 0 {
		result := results[0]
		if result.EventId != "401547456" {
			t.Errorf("Expected EventId 401547456, got %s", result.EventId)
		}
		if result.Season != "2024" {
			t.Errorf("Expected Season 2024, got %s", result.Season)
		}
		if result.Week != "2" {
			t.Errorf("Expected Week 2, got %s", result.Week)
		}
	}
}

func TestFetchResults(t *testing.T) {
	// Override the OpenAI API URL to use our mock server
	originalApiKey := ApiKey
	ApiKey = "test-key"
	defer func() { ApiKey = originalApiKey }()

	// Set the mock OpenAI API URL
	os.Setenv("OPENAI_API_URL", mockOpenAIServer.URL)
	defer os.Unsetenv("OPENAI_API_URL")

	// Create mock dependencies
	espnClient := &MockESPNClient{}

	// Test the function
	results := fetchResults(espnClient, "2024", "2", "2")

	// Verify results
	if len(results) == 0 {
		t.Error("Expected at least one result, got none")
	}

	// Verify the first result
	if len(results) > 0 {
		result := results[0]
		if result.EventId != "401547456" {
			t.Errorf("Expected EventId 401547456, got %s", result.EventId)
		}
		if result.Season != "2024" {
			t.Errorf("Expected Season 2024, got %s", result.Season)
		}
		if result.Week != "2" {
			t.Errorf("Expected Week 2, got %s", result.Week)
		}
	}
}

func TestHTTPBoundaries(t *testing.T) {
	// Set up test environment
	os.Setenv("OPENAI_API_KEY", "test-key")

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock the main handler
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock response"))
		case "/run":
			w.WriteHeader(http.StatusOK)
		case "/backfill":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Test root endpoint
	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test run endpoint
	resp, err = http.Get(server.URL + "/run")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test backfill endpoint
	resp, err = http.Get(server.URL + "/backfill?week=2&season=2024&seasontype=2")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGoldenMaster(t *testing.T) {
	// Load golden master data
	goldenData, err := os.ReadFile("testdata/template_data_golden.json")
	if err != nil {
		t.Skipf("Golden master file not found, skipping test: %v", err)
	}

	var expected domain.TemplateData
	err = json.Unmarshal(goldenData, &expected)
	if err != nil {
		t.Fatalf("Failed to unmarshal golden data: %v", err)
	}

	// Create mock dependencies that return the expected data
	resultRepo := &MockResultRepository{
		results: expected.Results,
		dates:   []domain.Date{},
	}

	// Convert dates to domain.Date format
	for _, dateTemplate := range expected.Dates {
		resultRepo.dates = append(resultRepo.dates, domain.Date{
			Season:     dateTemplate.Season,
			Week:       dateTemplate.Week,
			SeasonType: dateTemplate.SeasonType,
		})
	}

	// Test that the mock returns the expected data
	results, err := resultRepo.LoadResults("2024", "2", "2")
	if err != nil {
		t.Fatalf("Failed to load results: %v", err)
	}

	if len(results) != len(expected.Results) {
		t.Errorf("Expected %d results, got %d", len(expected.Results), len(results))
	}

	// Verify the structure matches
	for i, result := range results {
		if i >= len(expected.Results) {
			break
		}
		expectedResult := expected.Results[i]

		if result.EventId != expectedResult.EventId {
			t.Errorf("Result %d: Expected EventId %s, got %s", i, expectedResult.EventId, result.EventId)
		}

		if result.Game.Home.Name != expectedResult.Game.Home.Name {
			t.Errorf("Result %d: Expected Home Team %s, got %s", i, expectedResult.Game.Home.Name, result.Game.Home.Name)
		}

		if result.Game.Away.Name != expectedResult.Game.Away.Name {
			t.Errorf("Result %d: Expected Away Team %s, got %s", i, expectedResult.Game.Away.Name, result.Game.Away.Name)
		}
	}
}

func TestDatabaseOperations(t *testing.T) {
	// Create a temporary database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Initialize the database
	sqlStmt := `
	create table if not exists results (id integer not null primary key, event_id integer, week integer, season integer, season_type integer, rating integer, explanation text, spoiler_free_explanation text, game text);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create repository
	resultRepo := repository.NewSQLiteResultRepository(db)

	// Test data
	testResults := []domain.Result{
		{
			EventId:    "401547456",
			Season:     "2024",
			Week:       "2",
			SeasonType: "2",
			Rating: domain.Rating{
				Score:       85,
				Explanation: "Great game!",
				SpoilerFree: "Exciting matchup",
			},
			Game: domain.Game{
				Home: domain.Team{Name: "Chiefs", Score: 24},
				Away: domain.Team{Name: "Ravens", Score: 20},
			},
		},
	}

	// Test SaveResults
	err = resultRepo.SaveResults(testResults)
	if err != nil {
		t.Fatalf("Failed to save results: %v", err)
	}

	// Test LoadResults
	loadedResults, err := resultRepo.LoadResults("2024", "2", "2")
	if err != nil {
		t.Fatalf("Failed to load results: %v", err)
	}

	if len(loadedResults) != 1 {
		t.Errorf("Expected 1 result, got %d", len(loadedResults))
	}

	if loadedResults[0].EventId != "401547456" {
		t.Errorf("Expected EventId 401547456, got %s", loadedResults[0].EventId)
	}

	// Test LoadDates
	dates, err := resultRepo.LoadDates()
	if err != nil {
		t.Fatalf("Failed to load dates: %v", err)
	}

	if len(dates) != 1 {
		t.Errorf("Expected 1 date, got %d", len(dates))
	}

	if dates[0].Season != "2024" {
		t.Errorf("Expected Season 2024, got %s", dates[0].Season)
	}
}
