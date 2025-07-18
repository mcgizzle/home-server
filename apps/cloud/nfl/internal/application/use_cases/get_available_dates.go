package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// GetAvailableDatesUseCase defines the business operation for getting available dates
type GetAvailableDatesUseCase interface {
	Execute() ([]domain.Date, error)
}

// getAvailableDatesUseCase implements GetAvailableDatesUseCase
type getAvailableDatesUseCase struct {
	resultRepo repository.ResultRepository
}

// NewGetAvailableDatesUseCase creates a new instance of GetAvailableDatesUseCase
func NewGetAvailableDatesUseCase(resultRepo repository.ResultRepository) GetAvailableDatesUseCase {
	return &getAvailableDatesUseCase{
		resultRepo: resultRepo,
	}
}

// Execute gets all available dates from the repository
func (uc *getAvailableDatesUseCase) Execute() ([]domain.Date, error) {
	dates, err := uc.resultRepo.LoadDates()
	if err != nil {
		log.Printf("Error loading dates: %v", err)
		return nil, err
	}
	return dates, nil
}
