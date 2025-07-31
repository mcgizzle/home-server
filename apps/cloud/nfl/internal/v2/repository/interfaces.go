package repository

import (
	v1domain "github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
)

// LegacyResultRepository provides backward compatibility with V1 interfaces
// This allows gradual migration by implementing the existing ResultRepository interface
type LegacyResultRepository interface {
	// V1 compatibility methods - these coordinate between the separated repositories
	SaveResults(results []v1domain.Result) error
	LoadResults(season, week, seasonType string) ([]v1domain.Result, error)
	LoadDates() ([]v1domain.Date, error)
}

// DataSource provides abstraction for external data providers (ESPN, etc.)
type DataSource interface {
	GetSport() domain.Sport
	ListLatest() ([]domain.Competition, error)
	ListSpecific(season, period, periodType string) ([]domain.Competition, error)
	GetCompetition(id string) (domain.Competition, error)
}

// RatingService provides abstraction for rating generation (OpenAI, etc.)
type RatingService interface {
	ProduceRating(comp domain.Competition) (domain.Rating, error)
}
