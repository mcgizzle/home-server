package use_cases

import (
	"fmt"
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// FetchSpecificCompetitionsUseCase defines the V2 business operation for fetching specific period competitions
type FetchSpecificCompetitionsUseCase interface {
	Execute(sportID, season, period, periodType string) ([]domain.Competition, error)
	ExecuteWithLimit(sportID, season, period, periodType string, limit int) ([]domain.Competition, error)
	ExecuteForUpdate(sportID, season, period, periodType string) ([]domain.Competition, error)
	ExecuteForUpdateWithLimit(sportID, season, period, periodType string, limit int) ([]domain.Competition, error)
}

// fetchSpecificCompetitionsUseCase implements FetchSpecificCompetitionsUseCase
type fetchSpecificCompetitionsUseCase struct {
	sportsData      services.SportsDataService
	competitionRepo repository.CompetitionRepository
}

// NewFetchSpecificCompetitionsUseCase creates a new instance of V2 FetchSpecificCompetitionsUseCase
func NewFetchSpecificCompetitionsUseCase(
	sportsData services.SportsDataService,
	competitionRepo repository.CompetitionRepository,
) FetchSpecificCompetitionsUseCase {
	return &fetchSpecificCompetitionsUseCase{
		sportsData:      sportsData,
		competitionRepo: competitionRepo,
	}
}

// Execute fetches specific period competitions using the clean architecture interface
func (uc *fetchSpecificCompetitionsUseCase) Execute(sportID, season, period, periodType string) ([]domain.Competition, error) {
	return uc.ExecuteWithLimit(sportID, season, period, periodType, -1) // -1 means no limit
}

// ExecuteWithLimit fetches specific period competitions with a limit on how many to process
func (uc *fetchSpecificCompetitionsUseCase) ExecuteWithLimit(sportID, season, period, periodType string, limit int) ([]domain.Competition, error) {
	sport := domain.Sport(sportID)

	// Create date for the specific period
	date := domain.Date{
		Season:     season,
		Period:     period,
		PeriodType: periodType,
	}

	// Check for existing competitions to avoid duplicates
	existingCompetitions, err := uc.competitionRepo.FindByPeriod(season, period, periodType, sport)
	if err != nil {
		log.Printf("Error loading existing competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Get competitions for the specific period
	competitions, err := uc.sportsData.GetCompetitions(sport, date)
	if err != nil {
		log.Printf("Error getting competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Filter out competitions that already exist and apply limit
	var newCompetitions []domain.Competition
	processedCount := 0

	for _, comp := range competitions {
		// Check limit before processing each competition
		if limit > 0 && processedCount >= limit {
			log.Printf("Reached processing limit of %d competitions for %s %s %s, stopping early",
				limit, season, period, periodType)
			break
		}

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
			processedCount++
		}
	}

	limitMsg := ""
	if limit > 0 {
		limitMsg = fmt.Sprintf(" (limit: %d)", limit)
	}
	log.Printf("Fetched %d new competitions for specific period%s", len(newCompetitions), limitMsg)
	return newCompetitions, nil
}

// ExecuteForUpdate fetches ALL competitions for a period (including existing ones) for update purposes
func (uc *fetchSpecificCompetitionsUseCase) ExecuteForUpdate(sportID, season, period, periodType string) ([]domain.Competition, error) {
	return uc.ExecuteForUpdateWithLimit(sportID, season, period, periodType, -1) // -1 means no limit
}

// ExecuteForUpdateWithLimit fetches ALL competitions for a period with a limit, ignoring existing competitions
func (uc *fetchSpecificCompetitionsUseCase) ExecuteForUpdateWithLimit(sportID, season, period, periodType string, limit int) ([]domain.Competition, error) {
	sport := domain.Sport(sportID)

	// Create date for the specific period
	date := domain.Date{
		Season:     season,
		Period:     period,
		PeriodType: periodType,
	}

	// Get competitions for the specific period (don't filter out existing ones for updates)
	competitions, err := uc.sportsData.GetCompetitions(sport, date)
	if err != nil {
		log.Printf("Error getting competitions: %v", err)
		return []domain.Competition{}, err
	}

	// Apply limit if specified
	var limitedCompetitions []domain.Competition
	processedCount := 0

	for _, comp := range competitions {
		// Check limit before processing each competition
		if limit > 0 && processedCount >= limit {
			log.Printf("Reached processing limit of %d competitions for update %s %s %s, stopping early",
				limit, season, period, periodType)
			break
		}

		// Fetch details so we can fill missing play-by-play during update
		details, derr := uc.sportsData.GetCompetitionDetails(comp.ID)
		if derr != nil {
			log.Printf("Warning: failed to get details for %s: %v", comp.ID, derr)
		} else if details != nil {
			comp.Details = details
		}

		limitedCompetitions = append(limitedCompetitions, comp)
		processedCount++
	}

	limitMsg := ""
	if limit > 0 {
		limitMsg = fmt.Sprintf(" (limit: %d)", limit)
	}
	log.Printf("Fetched %d competitions for update%s", len(limitedCompetitions), limitMsg)
	return limitedCompetitions, nil
}
