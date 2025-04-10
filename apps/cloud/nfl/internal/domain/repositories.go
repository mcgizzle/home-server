package domain

import "context"

// GameRepository defines the interface for game data persistence
type GameRepository interface {
	// SaveGame saves a game to the repository
	SaveGame(ctx context.Context, game *Game) error

	// GetGame retrieves a game by ID
	GetGame(ctx context.Context, id string) (*Game, error)

	// ListGames retrieves all games for a given date
	ListGames(ctx context.Context, season string, week string, seasonType SeasonType) ([]*Game, error)

	// SaveRating saves a game rating
	SaveRating(ctx context.Context, gameID string, rating *Rating) error

	// GetRating retrieves a rating for a game
	GetRating(ctx context.Context, gameID string) (*Rating, error)
}
