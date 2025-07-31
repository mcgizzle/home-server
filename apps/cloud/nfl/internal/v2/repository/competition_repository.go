package repository

import (
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
)

// CompetitionRepository handles competition-specific operations
type CompetitionRepository interface {
	SaveCompetition(comp domain.Competition) error
	FindByPeriod(season, period, periodType string, sport domain.Sport) ([]domain.Competition, error)
	GetAvailablePeriods(sport domain.Sport) ([]domain.Date, error)
}
