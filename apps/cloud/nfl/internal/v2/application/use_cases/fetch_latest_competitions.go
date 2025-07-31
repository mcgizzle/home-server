package use_cases

import (
	"log"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"
)

// FetchLatestCompetitionsUseCase defines the V2 business operation for fetching latest competitions
type FetchLatestCompetitionsUseCase interface {
	Execute(sportID string) ([]domain.Competition, error)
}

// fetchLatestCompetitionsUseCase implements FetchLatestCompetitionsUseCase
type fetchLatestCompetitionsUseCase struct {
	espnClient      external.ESPNClient
	competitionRepo repository.CompetitionRepository
}

// NewFetchLatestCompetitionsUseCase creates a new instance of V2 FetchLatestCompetitionsUseCase
func NewFetchLatestCompetitionsUseCase(
	espnClient external.ESPNClient,
	competitionRepo repository.CompetitionRepository,
) FetchLatestCompetitionsUseCase {
	return &fetchLatestCompetitionsUseCase{
		espnClient:      espnClient,
		competitionRepo: competitionRepo,
	}
}

// Execute fetches the latest competitions from ESPN and processes them into V2 entities
func (uc *fetchLatestCompetitionsUseCase) Execute(sportID string) ([]domain.Competition, error) {
	sport := domain.Sport(sportID)

	// Get latest events from ESPN (for now, assumes NFL)
	eventRefs, err := uc.espnClient.ListLatestEvents()
	if err != nil {
		log.Printf("Error listing latest events: %v", err)
		return []domain.Competition{}, err
	}

	season := eventRefs.Meta.Parameters.Season[0]
	period := eventRefs.Meta.Parameters.Week[0]
	seasonType := eventRefs.Meta.Parameters.SeasonTypes[0]

	// Convert season type to V2 period type
	periodType := domain.PeriodTypeRegular
	if seasonType == "3" {
		periodType = domain.PeriodTypePlayoff
	}

	// Check for existing competitions to avoid duplicates
	existingCompetitions, err := uc.competitionRepo.FindByPeriod(season, period, periodType, sport)
	if err != nil {
		log.Printf("Error loading existing competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Filter out events that have already been processed
	var filteredEventRefs []external.EventRef
	for _, eventRef := range eventRefs.Items {
		event, err := uc.espnClient.GetEvent(eventRef.Ref)
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
			filteredEventRefs = append(filteredEventRefs, eventRef)
		}
	}

	var competitions []domain.Competition
	for _, eventRef := range filteredEventRefs {
		log.Printf("Processing competition: Season %s - Period %s - Sport %s", season, period, sportID)

		event, err := uc.espnClient.GetEvent(eventRef.Ref)
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
	}

	log.Printf("Fetched %d V2 competitions (no ratings)", len(competitions))
	return competitions, nil
}

// createCompetitionFromESPNEvent creates a V2 Competition directly from ESPN API data
func (uc *fetchLatestCompetitionsUseCase) createCompetitionFromESPNEvent(
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

// Helper functions

func generateTeamID(teamName string, sport domain.Sport) string {
	return string(sport) + "_" + teamName
}

func generateCompetitionID(eventID string) string {
	return "comp_" + eventID
}

func getTeamLogoURL(teamResp external.TeamResponse) string {
	if len(teamResp.Logos) > 0 {
		return teamResp.Logos[0].Href
	}
	return ""
}
