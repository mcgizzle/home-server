package use_cases

import (
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// GetLatestRatedDateUseCase finds the most recent period that has at least one rated competition.
type GetLatestRatedDateUseCase interface {
	Execute(sportID string) (date domain.Date, found bool, err error)
}

type getLatestRatedDateUseCase struct {
	competitionRepo repository.CompetitionRepository
}

func NewGetLatestRatedDateUseCase(competitionRepo repository.CompetitionRepository) GetLatestRatedDateUseCase {
	return &getLatestRatedDateUseCase{competitionRepo: competitionRepo}
}

func (uc *getLatestRatedDateUseCase) Execute(sportID string) (domain.Date, bool, error) {
	sport := domain.Sport(sportID)
	dates, err := uc.competitionRepo.GetAvailablePeriods(sport)
	if err != nil {
		return domain.Date{}, false, err
	}
	// iterate from newest to oldest
	for i := len(dates) - 1; i >= 0; i-- {
		d := dates[i]
		comps, err := uc.competitionRepo.FindByPeriod(d.Season, d.Period, d.PeriodType, sport)
		if err != nil {
			continue
		}
		for _, c := range comps {
			if c.Rating != nil && c.Rating.Score > 0 {
				return d, true, nil
			}
		}
	}
	return domain.Date{}, false, nil
}
