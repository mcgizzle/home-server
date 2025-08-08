package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// SaveCompetitionsUseCase defines the V2 business operation for saving competitions
type SaveCompetitionsUseCase interface {
	Execute(competitions []domain.Competition) error
}

// saveCompetitionsUseCase implements SaveCompetitionsUseCase
type saveCompetitionsUseCase struct {
	competitionRepo repository.CompetitionRepository
}

// NewSaveCompetitionsUseCase creates a new instance of V2 SaveCompetitionsUseCase
func NewSaveCompetitionsUseCase(competitionRepo repository.CompetitionRepository) SaveCompetitionsUseCase {
	return &saveCompetitionsUseCase{
		competitionRepo: competitionRepo,
	}
}

// Execute saves V2 competitions using V2 repository
func (uc *saveCompetitionsUseCase) Execute(competitions []domain.Competition) error {
	for _, competition := range competitions {
		err := uc.competitionRepo.SaveCompetition(competition)
		if err != nil {
			log.Printf("Error saving competition %s: %v", competition.ID, err)
			return err
		}
	}

	log.Printf("Saved %d competitions using V2 repository", len(competitions))
	return nil
}
