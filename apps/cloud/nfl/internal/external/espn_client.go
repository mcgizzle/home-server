package external

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// ESPNClient defines the interface for ESPN API operations
type ESPNClient interface {
	ListLatestEvents() (LatestEvents, error)
	ListSpecificEvents(season, week, seasonType string) (SpecificEvents, error)
	GetEvent(ref string) (EventResponse, error)
	GetEventById(id string) (EventResponse, error)
	GetScore(ref string) (ScoreResponse, error)
	GetTeam(ref string) (TeamResponse, error)
	GetRecord(ref string) (RecordResponse, error)
	GetDetails(ref string, page int) (DetailsResponse, error)
	GetDetailsPaged(ref string) ([]DetailsResponse, error)
}

// HTTPESPNClient implements ESPNClient using HTTP requests
type HTTPESPNClient struct {
	client *http.Client
}

// NewHTTPESPNClient creates a new HTTP-based ESPN client
func NewHTTPESPNClient() *HTTPESPNClient {
	return &HTTPESPNClient{
		client: http.DefaultClient,
	}
}

// EventRef represents an ESPN event reference
type EventRef struct {
	Ref string `json:"$ref"`
}

// LatestEvents represents the latest events response from ESPN
type LatestEvents struct {
	Items []EventRef `json:"items"`
	Meta  struct {
		Parameters struct {
			Week        []string `json:"week"`
			Season      []string `json:"season"`
			SeasonTypes []string `json:"seasontypes"`
		} `json:"parameters"`
	} `json:"$meta"`
}

// EventId represents an ESPN event ID
type EventId struct {
	Id string `json:"id"`
}

// SpecificEvents represents specific events response from ESPN
type SpecificEvents struct {
	Events []EventId `json:"events"`
}

// EventResponse represents a complete event response from ESPN
type EventResponse struct {
	Id           string         `json:"id"`
	Competitions []Competitions `json:"competitions"`
}

// Competitions represents game competitions
type Competitions struct {
	Competitors   []Competitors `json:"competitors"`
	DetailsRefs   DetailsRef    `json:"details"`
	LiveAvailable bool          `json:"liveAvailable"`
}

// Competitors represents team competitors
type Competitors struct {
	Id       string    `json:"id"`
	Team     TeamRef   `json:"team"`
	Score    ScoreRef  `json:"score"`
	HomeAway string    `json:"homeAway"`
	Record   RecordRef `json:"record"`
	Stats    StatsRef  `json:"statistics"`
}

// TeamRef represents a team reference
type TeamRef struct {
	Ref string `json:"$ref"`
}

// ScoreRef represents a score reference
type ScoreRef struct {
	Ref string `json:"$ref"`
}

// RecordRef represents a record reference
type RecordRef struct {
	Ref string `json:"$ref"`
}

// StatsRef represents a statistics reference
type StatsRef struct {
	Ref string `json:"$ref"`
}

// DetailsRef represents a details reference
type DetailsRef struct {
	Ref string `json:"$ref"`
}

// ScoreResponse represents a score response from ESPN
type ScoreResponse struct {
	Value float64 `json:"value"`
}

// TeamResponse represents a team response from ESPN
type TeamResponse struct {
	DisplayName string `json:"displayName"`
	Logos       []struct {
		Href string `json:"href"`
	} `json:"logos"`
}

// RecordResponse represents a record response from ESPN
type RecordResponse struct {
	Items []RecordItem `json:"items"`
}

// RecordItem represents a record item
type RecordItem struct {
	DisplayValue string `json:"displayValue"`
}

// DetailsItem represents individual plays or events in games
type DetailsItem struct {
	Text string `json:"text"`
}

// DetailsResponse represents a details response from ESPN
type DetailsResponse struct {
	PageIndex int           `json:"pageIndex"`
	PageCount int           `json:"pageCount"`
	Items     []DetailsItem `json:"items"`
}

// ListLatestEvents fetches data for the current week's games
func (c *HTTPESPNClient) ListLatestEvents() (LatestEvents, error) {
	req, err := http.NewRequest("GET", "https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events", nil)
	if err != nil {
		return LatestEvents{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return LatestEvents{}, err
	}
	defer response.Body.Close()

	var res LatestEvents
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return LatestEvents{}, err
	}

	return res, nil
}

// ListSpecificEvents fetches specific events for a season, week, and season type
func (c *HTTPESPNClient) ListSpecificEvents(season, week, seasonType string) (SpecificEvents, error) {
	url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard?week=%s&dates=%s&seasontype=%s", week, season, seasonType)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return SpecificEvents{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return SpecificEvents{}, err
	}
	defer response.Body.Close()

	var res SpecificEvents
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return SpecificEvents{}, err
	}

	return res, nil
}

// GetEvent fetches an event by its reference URL
func (c *HTTPESPNClient) GetEvent(ref string) (EventResponse, error) {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return EventResponse{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return EventResponse{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Error closing response body: %s", err)
		}
	}(response.Body)

	var res EventResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return EventResponse{}, err
	}

	return res, nil
}

// GetEventById fetches an event by its ID
func (c *HTTPESPNClient) GetEventById(id string) (EventResponse, error) {
	url := fmt.Sprintf("https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/events/%s", id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return EventResponse{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return EventResponse{}, err
	}
	defer response.Body.Close()

	var res EventResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return EventResponse{}, err
	}

	return res, nil
}

// GetScore fetches a score by its reference URL
func (c *HTTPESPNClient) GetScore(ref string) (ScoreResponse, error) {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return ScoreResponse{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return ScoreResponse{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(response.Body)

	var res ScoreResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return ScoreResponse{}, err
	}

	return res, nil
}

// GetTeam fetches a team by its reference URL
func (c *HTTPESPNClient) GetTeam(ref string) (TeamResponse, error) {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return TeamResponse{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return TeamResponse{}, err
	}
	defer response.Body.Close()

	var res TeamResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return TeamResponse{}, err
	}

	return res, nil
}

// GetRecord fetches a record by its reference URL
func (c *HTTPESPNClient) GetRecord(ref string) (RecordResponse, error) {
	req, err := http.NewRequest("GET", ref, nil)
	if err != nil {
		return RecordResponse{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return RecordResponse{}, err
	}
	defer response.Body.Close()

	var res RecordResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&res)
	if err != nil {
		return RecordResponse{}, err
	}

	return res, nil
}

// GetDetails fetches details by reference URL and page number
func (c *HTTPESPNClient) GetDetails(ref string, page int) (DetailsResponse, error) {
	url := fmt.Sprintf("%s&page=%d", ref, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return DetailsResponse{}, err
	}

	response, err := c.client.Do(req)
	if err != nil {
		return DetailsResponse{}, err
	}
	defer response.Body.Close()

	var res DetailsResponse
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return DetailsResponse{}, err
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		return DetailsResponse{}, err
	}

	return res, nil
}

// GetDetailsPaged fetches all pages of details
func (c *HTTPESPNClient) GetDetailsPaged(ref string) ([]DetailsResponse, error) {
	var details []DetailsResponse

	res, err := c.GetDetails(ref, 1)
	if err != nil {
		return nil, err
	}

	details = append(details, res)
	for i := 2; i <= res.PageCount; i++ {
		res, err := c.GetDetails(ref, i)
		if err != nil {
			return nil, err
		}
		details = append(details, res)
	}
	return details, nil
}
