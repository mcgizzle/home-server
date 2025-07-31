package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"
)

// GetAvailableDatesUseCase defines the V2 business operation for getting available dates
type GetAvailableDatesUseCase interface {
	Execute(sportID string) ([]domain.Date, error)
}

// getAvailableDatesUseCase implements GetAvailableDatesUseCase
type getAvailableDatesUseCase struct {
	competitionRepo repository.CompetitionRepository
}

// NewGetAvailableDatesUseCase creates a new instance of V2 GetAvailableDatesUseCase
func NewGetAvailableDatesUseCase(competitionRepo repository.CompetitionRepository) GetAvailableDatesUseCase {
	return &getAvailableDatesUseCase{
		competitionRepo: competitionRepo,
	}
}

// Execute gets all available dates from the repository for a specific sport
func (uc *getAvailableDatesUseCase) Execute(sportID string) ([]domain.Date, error) {
	sport := domain.Sport(sportID)
	dates, err := uc.competitionRepo.GetAvailablePeriods(sport)
	if err != nil {
		log.Printf("Error loading dates for sport %s: %v", sportID, err)
		return nil, err
	}

	log.Printf("Successfully loaded %d available dates for sport %s", len(dates), sportID)
	return dates, nil
}
