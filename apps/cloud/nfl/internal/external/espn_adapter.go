package external

import (
	"fmt"
	"log"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// ESPNAdapter implements the SportsDataService interface using ESPN API
// This adapter converts ESPN-specific responses to domain entities
type ESPNAdapter struct {
	client ESPNClient
}

// NewESPNAdapter creates a new ESPN adapter that implements SportsDataService
func NewESPNAdapter(client ESPNClient) services.SportsDataService {
	return &ESPNAdapter{
		client: client,
	}
}

// GetAvailablePeriods returns all available periods for a sport and season
func (a *ESPNAdapter) GetAvailablePeriods(sport domain.Sport, season string) ([]domain.Date, error) {
	// For now, only support NFL
	if sport != "nfl" {
		return nil, fmt.Errorf("sport %s not supported", sport)
	}

	// Get latest events to extract available periods
	latestEvents, err := a.client.ListLatestEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest events: %w", err)
	}

	var dates []domain.Date
	seenDates := make(map[string]bool)

	// Extract unique combinations of season, period, and period type
	for _, week := range latestEvents.Meta.Parameters.Week {
		for _, seasonParam := range latestEvents.Meta.Parameters.Season {
			for _, seasonType := range latestEvents.Meta.Parameters.SeasonTypes {
				// Filter by requested season if provided
				if season != "" && seasonParam != season {
					continue
				}

				dateKey := fmt.Sprintf("%s-%s-%s", seasonParam, week, seasonType)
				if !seenDates[dateKey] {
					dates = append(dates, domain.Date{
						Season:     seasonParam,
						Period:     week,
						PeriodType: mapESPNSeasonTypeToDomain(seasonType),
					})
					seenDates[dateKey] = true
				}
			}
		}
	}

	return dates, nil
}

// GetLatest returns the latest/current period information for a sport
func (a *ESPNAdapter) GetLatest(sport domain.Sport) (*domain.Date, error) {
	// For now, only support NFL
	if sport != "nfl" {
		return nil, fmt.Errorf("sport %s not supported", sport)
	}

	// Get latest events to extract the current period
	latestEvents, err := a.client.ListLatestEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest events: %w", err)
	}

	// ESPN's ListLatestEvents returns the current active period
	// We use the first values from the parameters as they represent the "current" period
	if len(latestEvents.Meta.Parameters.Season) == 0 ||
		len(latestEvents.Meta.Parameters.Week) == 0 ||
		len(latestEvents.Meta.Parameters.SeasonTypes) == 0 {
		return nil, fmt.Errorf("no current period information available")
	}

	season := latestEvents.Meta.Parameters.Season[0]
	period := latestEvents.Meta.Parameters.Week[0]
	seasonType := latestEvents.Meta.Parameters.SeasonTypes[0]

	return &domain.Date{
		Season:     season,
		Period:     period,
		PeriodType: mapESPNSeasonTypeToDomain(seasonType),
	}, nil
}

// GetCompetitions retrieves all competitions for a specific period
func (a *ESPNAdapter) GetCompetitions(sport domain.Sport, date domain.Date) ([]domain.Competition, error) {
	// For now, only support NFL
	if sport != "nfl" {
		return nil, fmt.Errorf("sport %s not supported", sport)
	}

	// Convert domain period type back to ESPN season type
	espnSeasonType := mapDomainPeriodTypeToESPN(date.PeriodType)

	// Get specific events for the period
	specificEvents, err := a.client.ListSpecificEvents(date.Season, date.Period, espnSeasonType)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for period: %w", err)
	}

	var competitions []domain.Competition
	for _, eventId := range specificEvents.Events {
		// Get detailed event data
		event, err := a.client.GetEventById(eventId.Id)
		if err != nil {
			log.Printf("Failed to get event %s: %v", eventId.Id, err)
			continue
		}

		// Convert ESPN event to domain competition
		competition, err := a.convertEventToCompetition(event, sport, date)
		if err != nil {
			log.Printf("Failed to convert event %s: %v", eventId.Id, err)
			continue
		}

		competitions = append(competitions, *competition)
	}

	return competitions, nil
}

// GetCompetitionDetails retrieves detailed competition data including play-by-play
func (a *ESPNAdapter) GetCompetitionDetails(competitionID string) (*domain.CompetitionDetails, error) {
	// Get event to find details reference
	event, err := a.client.GetEventById(competitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if len(event.Competitions) == 0 {
		return nil, fmt.Errorf("no competitions found for event %s", competitionID)
	}

	// Get paginated details
	detailsRef := event.Competitions[0].DetailsRefs.Ref
	if detailsRef == "" {
		return nil, fmt.Errorf("missing details ref for competition %s", competitionID)
	}
	detailsResponses, err := a.client.GetDetailsPaged(detailsRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get competition details: %w", err)
	}

	// Convert to domain format
	var playByPlay []interface{}
	for _, detailsResponse := range detailsResponses {
		for _, item := range detailsResponse.Items {
			playByPlay = append(playByPlay, map[string]interface{}{
				"text": item.Text,
			})
		}
	}

	return &domain.CompetitionDetails{
		PlayByPlay: playByPlay,
		Metadata: map[string]interface{}{
			"source": "espn",
			"pages":  len(detailsResponses),
		},
	}, nil
}

// GetTeam retrieves team information by team ID
func (a *ESPNAdapter) GetTeam(teamID string) (*domain.Team, error) {
	// Construct ESPN team URL - this might need adjustment based on ESPN's actual URL structure
	teamURL := fmt.Sprintf("https://sports.core.api.espn.com/v2/sports/football/leagues/nfl/teams/%s", teamID)

	teamResponse, err := a.client.GetTeam(teamURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	logoURL := ""
	if len(teamResponse.Logos) > 0 {
		logoURL = teamResponse.Logos[0].Href
	}

	return &domain.Team{
		ID:      teamID,
		Name:    teamResponse.DisplayName,
		Sport:   "nfl",
		LogoURL: logoURL,
	}, nil
}

// Helper functions

func (a *ESPNAdapter) convertEventToCompetition(event EventResponse, sport domain.Sport, date domain.Date) (*domain.Competition, error) {
	if len(event.Competitions) == 0 {
		return nil, fmt.Errorf("no competitions found in event")
	}

	espnCompetition := event.Competitions[0]

	// Skip live games
	if espnCompetition.LiveAvailable {
		return nil, fmt.Errorf("skipping live game")
	}

	var teams []domain.CompetitionTeam
	for _, competitor := range espnCompetition.Competitors {
		// Get team details
		team, err := a.client.GetTeam(competitor.Team.Ref)
		if err != nil {
			return nil, fmt.Errorf("failed to get team: %w", err)
		}

		// Get score
		score, err := a.client.GetScore(competitor.Score.Ref)
		if err != nil {
			return nil, fmt.Errorf("failed to get score: %w", err)
		}

		logoURL := ""
		if len(team.Logos) > 0 {
			logoURL = team.Logos[0].Href
		}

		domainTeam := domain.Team{
			ID:      competitor.Id,
			Name:    team.DisplayName,
			Sport:   sport,
			LogoURL: logoURL,
		}

		competitionTeam := domain.CompetitionTeam{
			Team:     domainTeam,
			HomeAway: competitor.HomeAway,
			Score:    score.Value,
			Stats:    make(map[string]interface{}),
		}

		teams = append(teams, competitionTeam)
	}

	// TODO: This is a hack to skip games that haven't been played (score is 0 for both teams)
	// Skip games that haven't been played (score is 0 for both teams)
	if len(teams) >= 2 && teams[0].Score == 0 && teams[1].Score == 0 {
		return nil, fmt.Errorf("skipping unplayed game")
	}

	// Parse the start time from ESPN competition date
	var startTime *time.Time
	if espnCompetition.Date != "" {
		// ESPN uses ISO 8601 format with 'Z' suffix for UTC
		// Try RFC3339 first, then fall back to custom format
		if parsedTime, err := time.Parse(time.RFC3339, espnCompetition.Date); err == nil {
			startTime = &parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02T15:04Z", espnCompetition.Date); err == nil {
			startTime = &parsedTime
		} else {
			log.Printf("Failed to parse competition date %s: %v", espnCompetition.Date, err)
		}
	}

	return &domain.Competition{
		ID:         event.Id,
		EventID:    event.Id,
		Sport:      sport,
		Season:     date.Season,
		Period:     date.Period,
		PeriodType: date.PeriodType,
		StartTime:  startTime,
		Status:     "completed", // Could be enhanced to detect actual status
		Teams:      teams,
		CreatedAt:  time.Now(),
	}, nil
}

// mapESPNSeasonTypeToDomain converts ESPN season type numbers to domain period types
func mapESPNSeasonTypeToDomain(espnSeasonType string) string {
	switch espnSeasonType {
	case "1":
		return "preseason"
	case "2":
		return "regular"
	case "3":
		return "playoff"
	default:
		return "regular"
	}
}

// mapDomainPeriodTypeToESPN converts domain period types back to ESPN season type numbers
func mapDomainPeriodTypeToESPN(periodType string) string {
	switch periodType {
	case "preseason":
		return "1"
	case "regular":
		return "2"
	case "playoff":
		return "3"
	default:
		return "2"
	}
}
