package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// FillMissingDetailsUseCase defines the operation to populate play-by-play details
// for competitions that are missing them across recent periods.
type FillMissingDetailsUseCase interface {
	Execute(sportID string, recentPeriods int) (processed int, updated int, err error)
}

type fillMissingDetailsUseCase struct {
	sportsData      services.SportsDataService
	competitionRepo repository.CompetitionRepository
}

func NewFillMissingDetailsUseCase(
	sportsData services.SportsDataService,
	competitionRepo repository.CompetitionRepository,
) FillMissingDetailsUseCase {
	return &fillMissingDetailsUseCase{
		sportsData:      sportsData,
		competitionRepo: competitionRepo,
	}
}

func (uc *fillMissingDetailsUseCase) Execute(sportID string, recentPeriods int) (int, int, error) {
	processed := 0
	updated := 0

	dates, err := uc.competitionRepo.GetAvailablePeriods(domain.Sport(sportID))
	if err != nil {
		return 0, 0, err
	}
	if len(dates) == 0 {
		return 0, 0, nil
	}

	// Choose trailing window of periods
	start := 0
	if recentPeriods > 0 && len(dates) > recentPeriods {
		start = len(dates) - recentPeriods
	}
	recent := dates[start:]

	// Iterate newest first to prioritize latest data
	for i := len(recent) - 1; i >= 0; i-- {
		d := recent[i]
		comps, err := uc.competitionRepo.FindByPeriod(d.Season, d.Period, d.PeriodType, domain.Sport(sportID))
		if err != nil {
			log.Printf("Error loading competitions for %s %s %s: %v", d.Season, d.Period, d.PeriodType, err)
			continue
		}
		for _, c := range comps {
			processed++
			if c.Details != nil && len(c.Details.PlayByPlay) > 0 {
				continue
			}
			details, derr := uc.sportsData.GetCompetitionDetails(c.ID)
			if derr != nil {
				log.Printf("Details fetch failed for %s: %v", c.ID, derr)
				continue
			}
			if details == nil || len(details.PlayByPlay) == 0 {
				continue
			}
			c.Details = details
			if err := uc.competitionRepo.SaveCompetition(c); err != nil {
				log.Printf("Failed saving details for %s: %v", c.ID, err)
				continue
			}
			updated++
			log.Printf("Filled details for competition %s (%s %s %s)", c.ID, d.Season, d.Period, d.PeriodType)
		}
	}

	return processed, updated, nil
}
