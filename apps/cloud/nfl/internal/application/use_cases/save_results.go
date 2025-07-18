package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// SaveResultsUseCase defines the business operation for saving NFL results
type SaveResultsUseCase interface {
	Execute(results []domain.Result) error
}

// saveResultsUseCase implements SaveResultsUseCase
type saveResultsUseCase struct {
	resultRepo repository.ResultRepository
}

// NewSaveResultsUseCase creates a new instance of SaveResultsUseCase
func NewSaveResultsUseCase(resultRepo repository.ResultRepository) SaveResultsUseCase {
	return &saveResultsUseCase{
		resultRepo: resultRepo,
	}
}

// Execute saves NFL results to the repository
func (uc *saveResultsUseCase) Execute(results []domain.Result) error {
	err := uc.resultRepo.SaveResults(results)
	if err != nil {
		log.Printf("Error saving results: %v", err)
		return err
	}
	log.Printf("Saved %d results", len(results))
	return nil
}
