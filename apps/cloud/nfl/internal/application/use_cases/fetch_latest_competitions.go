package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// FetchLatestCompetitionsUseCase defines the V2 business operation for fetching latest competitions
type FetchLatestCompetitionsUseCase interface {
	Execute(sportID string) ([]domain.Competition, error)
}

// fetchLatestCompetitionsUseCase implements FetchLatestCompetitionsUseCase
type fetchLatestCompetitionsUseCase struct {
	sportsData      services.SportsDataService
	competitionRepo repository.CompetitionRepository
}

// NewFetchLatestCompetitionsUseCase creates a new instance of V2 FetchLatestCompetitionsUseCase
func NewFetchLatestCompetitionsUseCase(
	sportsData services.SportsDataService,
	competitionRepo repository.CompetitionRepository,
) FetchLatestCompetitionsUseCase {
	return &fetchLatestCompetitionsUseCase{
		sportsData:      sportsData,
		competitionRepo: competitionRepo,
	}
}

// Execute fetches the latest competitions using the clean architecture interface
func (uc *fetchLatestCompetitionsUseCase) Execute(sportID string) ([]domain.Competition, error) {
	sport := domain.Sport(sportID)

	// Get the latest/current period for the sport
	latestDate, err := uc.sportsData.GetLatest(sport)
	if err != nil {
		log.Printf("Error getting latest period: %v", err)
		return []domain.Competition{}, err
	}

	if latestDate == nil {
		log.Printf("No current period found for sport %s", sportID)
		return []domain.Competition{}, nil
	}

	log.Printf("Using latest date: Season %s, Period %s, Type %s", latestDate.Season, latestDate.Period, latestDate.PeriodType)

	// Check for existing competitions to avoid duplicates
	existingCompetitions, err := uc.competitionRepo.FindByPeriod(latestDate.Season, latestDate.Period, latestDate.PeriodType, sport)
	if err != nil {
		log.Printf("Error loading existing competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Get competitions for the latest period
	competitions, err := uc.sportsData.GetCompetitions(sport, *latestDate)
	if err != nil {
		log.Printf("Error getting competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Filter out competitions that already exist
	var newCompetitions []domain.Competition
	for _, comp := range competitions {
		isExisting := false
		for _, existing := range existingCompetitions {
			if existing.EventID == comp.EventID {
				isExisting = true
				log.Printf("Competition already exists: %s", comp.EventID)
				break
			}
		}
		if !isExisting {
			// Fetch details so we persist play-by-play when saving
			details, derr := uc.sportsData.GetCompetitionDetails(comp.ID)
			if derr != nil {
				log.Printf("Warning: failed to get details for %s: %v", comp.ID, derr)
			} else if details != nil {
				comp.Details = details
			}
			newCompetitions = append(newCompetitions, comp)
		}
	}

	log.Printf("Fetched %d new competitions out of %d total", len(newCompetitions), len(competitions))
	return newCompetitions, nil
}
