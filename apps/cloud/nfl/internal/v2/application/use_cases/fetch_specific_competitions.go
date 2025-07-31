package use_cases

import (
	"fmt"
	"log"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"
)

// FetchSpecificCompetitionsUseCase defines the V2 business operation for fetching specific period competitions
type FetchSpecificCompetitionsUseCase interface {
	Execute(sportID, season, period, periodType string) ([]domain.Competition, error)
	ExecuteWithLimit(sportID, season, period, periodType string, limit int) ([]domain.Competition, error)
}

// fetchSpecificCompetitionsUseCase implements FetchSpecificCompetitionsUseCase
type fetchSpecificCompetitionsUseCase struct {
	espnClient      external.ESPNClient
	competitionRepo repository.CompetitionRepository
}

// NewFetchSpecificCompetitionsUseCase creates a new instance of V2 FetchSpecificCompetitionsUseCase
func NewFetchSpecificCompetitionsUseCase(
	espnClient external.ESPNClient,
	competitionRepo repository.CompetitionRepository,
) FetchSpecificCompetitionsUseCase {
	return &fetchSpecificCompetitionsUseCase{
		espnClient:      espnClient,
		competitionRepo: competitionRepo,
	}
}

// Execute fetches specific period competitions from ESPN and processes them into V2 entities
func (uc *fetchSpecificCompetitionsUseCase) Execute(sportID, season, period, periodType string) ([]domain.Competition, error) {
	return uc.ExecuteWithLimit(sportID, season, period, periodType, -1) // -1 means no limit
}

// ExecuteWithLimit fetches specific period competitions with a limit on how many to process
func (uc *fetchSpecificCompetitionsUseCase) ExecuteWithLimit(sportID, season, period, periodType string, limit int) ([]domain.Competition, error) {
	sport := domain.Sport(sportID)

	// Convert V2 periodType to V1 seasonType for ESPN client compatibility
	seasonType := "2" // regular season
	if periodType == "playoff" {
		seasonType = "3"
	}

	// Get specific events from ESPN
	eventRefs, err := uc.espnClient.ListSpecificEvents(season, period, seasonType)
	if err != nil {
		log.Printf("Error listing specific events: %v", err)
		return []domain.Competition{}, err
	}

	// Check for existing competitions to avoid duplicates
	existingCompetitions, err := uc.competitionRepo.FindByPeriod(season, period, periodType, sport)
	if err != nil {
		log.Printf("Error loading existing competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Filter out events that have already been processed
	var filteredEventRefs []external.EventId
	for _, eventId := range eventRefs.Events {
		event, err := uc.espnClient.GetEventById(eventId.Id)
		if err != nil {
			log.Printf("Error getting event: %v", err)
			continue
		}

		shouldInclude := true
		for _, existingComp := range existingCompetitions {
			if existingComp.EventID == event.Id {
				shouldInclude = false
				log.Printf("Competition already processed: %s - %s", season, period)
				break
			}
		}
		if shouldInclude {
			filteredEventRefs = append(filteredEventRefs, eventId)
		}
	}

	var competitions []domain.Competition
	processedCount := 0

	for _, eventId := range filteredEventRefs {
		// Check limit before processing each event
		if limit > 0 && processedCount >= limit {
			log.Printf("Reached processing limit of %d competitions for %s %s %s, stopping early",
				limit, season, period, periodType)
			break
		}

		log.Printf("Processing specific competition: Season %s - Period %s - Sport %s", season, period, sportID)

		event, err := uc.espnClient.GetEventById(eventId.Id)
		if err != nil {
			log.Printf("Error getting event: %v", err)
			continue
		}

		// Check if event has competitions (games)
		if len(event.Competitions) == 0 {
			log.Printf("Event has no competitions, skipping")
			continue
		}

		competition := event.Competitions[0] // Take the first competition

		// Check if game has been played (has competitors with scores)
		if len(competition.Competitors) < 2 {
			log.Printf("Competition doesn't have enough competitors, skipping")
			continue
		}

		// Create V2 Competition directly from ESPN data (NO RATING GENERATION)
		v2Competition, err := uc.createCompetitionFromESPNEvent(event.Id, sport, season, period, periodType, competition)
		if err != nil {
			log.Printf("Error creating competition from ESPN data: %v", err)
			continue
		}

		competitions = append(competitions, v2Competition)
		processedCount++
	}

	limitMsg := ""
	if limit > 0 {
		limitMsg = fmt.Sprintf(" (limit: %d)", limit)
	}
	log.Printf("Fetched %d V2 competitions for specific period%s (no ratings)", len(competitions), limitMsg)
	return competitions, nil
}

// createCompetitionFromESPNEvent creates a V2 Competition directly from ESPN API data
func (uc *fetchSpecificCompetitionsUseCase) createCompetitionFromESPNEvent(
	eventID string,
	sport domain.Sport,
	season, period, periodType string,
	competition external.Competitions,
) (domain.Competition, error) {
	var competitionTeams []domain.CompetitionTeam

	// Process each competitor
	for _, competitor := range competition.Competitors {
		// Get team info
		teamResp, err := uc.espnClient.GetTeam(competitor.Team.Ref)
		if err != nil {
			log.Printf("Error getting team info: %v", err)
			continue
		}

		// Get score
		scoreResp, err := uc.espnClient.GetScore(competitor.Score.Ref)
		if err != nil {
			log.Printf("Error getting score: %v", err)
			continue
		}

		// Get record
		recordResp, err := uc.espnClient.GetRecord(competitor.Record.Ref)
		if err != nil {
			log.Printf("Error getting record: %v", err)
			continue
		}

		var recordDisplay string
		if len(recordResp.Items) > 0 {
			recordDisplay = recordResp.Items[0].DisplayValue
		}

		// Create team
		team := domain.Team{
			ID:      generateTeamID(teamResp.DisplayName, sport),
			Name:    teamResp.DisplayName,
			Sport:   sport,
			LogoURL: getTeamLogoURL(teamResp),
		}

		// Determine home/away
		homeAway := domain.HomeAwayHome
		if competitor.HomeAway == "away" {
			homeAway = domain.HomeAwayAway
		}

		// Create competition team
		competitionTeam := domain.CompetitionTeam{
			Team:     team,
			HomeAway: homeAway,
			Score:    scoreResp.Value,
			Stats:    map[string]interface{}{"record": recordDisplay},
		}

		competitionTeams = append(competitionTeams, competitionTeam)
	}

	// Create competition
	v2Competition := domain.Competition{
		ID:         generateCompetitionID(eventID),
		EventID:    eventID,
		Sport:      sport,
		Season:     season,
		Period:     period,
		PeriodType: periodType,
		Status:     domain.StatusCompleted,
		Teams:      competitionTeams,
		CreatedAt:  time.Now(),
	}

	// Add details if available
	if competition.DetailsRefs.Ref != "" {
		detailsResponses, err := uc.espnClient.GetDetailsPaged(competition.DetailsRefs.Ref)
		if err == nil && len(detailsResponses) > 0 {
			var allDetails []interface{}
			for _, detailsResp := range detailsResponses {
				for _, item := range detailsResp.Items {
					allDetails = append(allDetails, item)
				}
			}
			if len(allDetails) > 0 {
				details := &domain.CompetitionDetails{
					PlayByPlay: allDetails,
				}
				v2Competition.Details = details
			}
		}
	}

	return v2Competition, nil
}
