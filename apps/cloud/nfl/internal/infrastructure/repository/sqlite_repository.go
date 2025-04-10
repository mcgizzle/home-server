package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"nfl/internal/domain"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepository implements the GameRepository interface using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, err
	}

	return &SQLiteRepository{db: db}, nil
}

func createTables(db *sql.DB) error {
	// Create results table
	_, err := db.Exec(`
		create table if not exists results (
			id integer not null primary key,
			event_id integer,
			week integer,
			season integer,
			season_type integer,
			rating integer,
			explanation text,
			spoiler_free_explanation text,
			game text
		)
	`)
	return err
}

// SaveGame implements GameRepository.SaveGame
func (r *SQLiteRepository) SaveGame(ctx context.Context, game *domain.Game) error {
	// Convert game to JSON
	gameJson, err := json.Marshal(game)
	if err != nil {
		return err
	}

	// Convert string values to integers
	week, err := strconv.Atoi(game.Week)
	if err != nil {
		return err
	}
	season, err := strconv.Atoi(game.Season)
	if err != nil {
		return err
	}
	seasonType, err := strconv.Atoi(string(game.SeasonType))
	if err != nil {
		return err
	}
	eventID, err := strconv.Atoi(game.ID)
	if err != nil {
		return err
	}

	// Insert or update the game
	_, err = r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO results (
			event_id, season, week, season_type, game
		) VALUES (?, ?, ?, ?, ?)
	`,
		eventID,
		season,
		week,
		seasonType,
		string(gameJson),
	)
	return err
}

// GetGame implements GameRepository.GetGame
func (r *SQLiteRepository) GetGame(ctx context.Context, id string) (*domain.Game, error) {
	eventID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var gameJson string
	err = r.db.QueryRowContext(ctx, `
		SELECT game
		FROM results
		WHERE event_id = ?
	`, eventID).Scan(&gameJson)
	if err != nil {
		return nil, err
	}

	var game domain.Game
	if err := json.Unmarshal([]byte(gameJson), &game); err != nil {
		return nil, err
	}

	return &game, nil
}

// ListGames implements GameRepository.ListGames
func (r *SQLiteRepository) ListGames(ctx context.Context, season string, week string, seasonType domain.SeasonType) ([]*domain.Game, error) {
	// Convert string values to integers
	weekInt, err := strconv.Atoi(week)
	if err != nil {
		return nil, err
	}
	seasonInt, err := strconv.Atoi(season)
	if err != nil {
		return nil, err
	}
	seasonTypeInt, err := strconv.Atoi(string(seasonType))
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT game
		FROM results
		WHERE season = ? AND week = ? AND season_type = ?
		ORDER BY rating DESC
	`, seasonInt, weekInt, seasonTypeInt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []*domain.Game
	for rows.Next() {
		var gameJson string
		if err := rows.Scan(&gameJson); err != nil {
			return nil, err
		}

		var game domain.Game
		if err := json.Unmarshal([]byte(gameJson), &game); err != nil {
			return nil, err
		}

		games = append(games, &game)
	}

	return games, nil
}

// SaveRating implements GameRepository.SaveRating
func (r *SQLiteRepository) SaveRating(ctx context.Context, gameID string, rating *domain.Rating) error {
	eventID, err := strconv.Atoi(gameID)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE results
		SET rating = ?,
			explanation = ?,
			spoiler_free_explanation = ?
		WHERE event_id = ?
	`,
		rating.Score,
		rating.Explanation,
		rating.SpoilerFree,
		eventID,
	)
	return err
}

// GetRating implements GameRepository.GetRating
func (r *SQLiteRepository) GetRating(ctx context.Context, gameID string) (*domain.Rating, error) {
	eventID, err := strconv.Atoi(gameID)
	if err != nil {
		return nil, err
	}

	var rating domain.Rating
	err = r.db.QueryRowContext(ctx, `
		SELECT rating, explanation, spoiler_free_explanation
		FROM results
		WHERE event_id = ?
	`, eventID).Scan(
		&rating.Score,
		&rating.Explanation,
		&rating.SpoilerFree,
	)
	if err != nil {
		return nil, err
	}

	return &rating, nil
}
