package services

import "github.com/mcgizzle/home-server/apps/cloud/internal/domain"

// SportsDataService defines the interface for retrieving sports competition data.
// This interface is owned by the application layer and implemented by infrastructure.
// It follows clean architecture principles by defining what the business logic needs,
// not how the external data source (ESPN) structures its responses.
type SportsDataService interface {
	// GetAvailablePeriods returns all available periods (weeks/rounds) for a sport and season
	GetAvailablePeriods(sport domain.Sport, season string) ([]domain.Date, error)

	// GetLatest returns the latest/current period information for a sport
	GetLatest(sport domain.Sport) (*domain.Date, error)

	// GetCompetitions retrieves all competitions for a specific period
	GetCompetitions(sport domain.Sport, date domain.Date) ([]domain.Competition, error)

	// GetCompetitionDetails retrieves detailed competition data including play-by-play
	GetCompetitionDetails(competitionID string) (*domain.CompetitionDetails, error)

	// GetTeam retrieves team information by team ID
	GetTeam(teamID string) (*domain.Team, error)
}
