package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"nfl/internal/domain"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepository implements the GameRepository interface using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// Game represents a game in the repository layer
type Game struct {
	ID         string
	Season     string
	Week       string
	SeasonType string
	HomeTeam   Team
	AwayTeam   Team
	Score      Score
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Team represents a team in the repository layer
type Team struct {
	ID   string
	Name string
	Logo string
}

// Score represents a game's score in the repository layer
type Score struct {
	Home int
	Away int
}

// Rating represents a game rating in the repository layer
type Rating struct {
	Score       int
	SpoilerFree string
	Explanation string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

// toDomainGame converts a repository Game to a domain Game
func toDomainGame(repoGame *Game) *domain.Game {
	var seasonType domain.SeasonType
	switch repoGame.SeasonType {
	case "1":
		seasonType = domain.SeasonTypePreseason
	case "2":
		seasonType = domain.SeasonTypeRegular
	case "3":
		seasonType = domain.SeasonTypePlayoffs
	default:
		seasonType = domain.SeasonType(repoGame.SeasonType)
	}

	return &domain.Game{
		ID:         repoGame.ID,
		Season:     repoGame.Season,
		Week:       repoGame.Week,
		SeasonType: seasonType,
		HomeTeam: domain.Team{
			ID:   repoGame.HomeTeam.ID,
			Name: repoGame.HomeTeam.Name,
			Logo: repoGame.HomeTeam.Logo,
		},
		AwayTeam: domain.Team{
			ID:   repoGame.AwayTeam.ID,
			Name: repoGame.AwayTeam.Name,
			Logo: repoGame.AwayTeam.Logo,
		},
		Score: domain.Score{
			Home: repoGame.Score.Home,
			Away: repoGame.Score.Away,
		},
	}
}

// toRepositoryGame converts a domain Game to a repository Game
func toRepositoryGame(domainGame *domain.Game) *Game {
	return &Game{
		ID:         domainGame.ID,
		Season:     domainGame.Season,
		Week:       domainGame.Week,
		SeasonType: string(domainGame.SeasonType),
		HomeTeam: Team{
			ID:   domainGame.HomeTeam.ID,
			Name: domainGame.HomeTeam.Name,
			Logo: domainGame.HomeTeam.Logo,
		},
		AwayTeam: Team{
			ID:   domainGame.AwayTeam.ID,
			Name: domainGame.AwayTeam.Name,
			Logo: domainGame.AwayTeam.Logo,
		},
		Score: Score{
			Home: domainGame.Score.Home,
			Away: domainGame.Score.Away,
		},
		UpdatedAt: time.Now(),
	}
}

// toDomainRating converts a repository Rating to a domain Rating
func toDomainRating(repoRating *Rating) *domain.Rating {
	return &domain.Rating{
		Score:       repoRating.Score,
		SpoilerFree: repoRating.SpoilerFree,
		Explanation: repoRating.Explanation,
	}
}

// toRepositoryRating converts a domain Rating to a repository Rating
func toRepositoryRating(domainRating *domain.Rating) *Rating {
	return &Rating{
		Score:       domainRating.Score,
		SpoilerFree: domainRating.SpoilerFree,
		Explanation: domainRating.Explanation,
		UpdatedAt:   time.Now(),
	}
}

// SaveGame implements GameRepository.SaveGame
func (r *SQLiteRepository) SaveGame(ctx context.Context, domainGame *domain.Game) error {
	// Convert domain game to repository game
	repoGame := toRepositoryGame(domainGame)

	// Convert repository game to JSON
	gameJson, err := json.Marshal(repoGame)
	if err != nil {
		return err
	}

	// Convert string values to integers
	week, err := strconv.Atoi(repoGame.Week)
	if err != nil {
		return err
	}
	season, err := strconv.Atoi(repoGame.Season)
	if err != nil {
		return err
	}
	seasonType, err := strconv.Atoi(repoGame.SeasonType)
	if err != nil {
		return err
	}
	eventID, err := strconv.Atoi(repoGame.ID)
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

	var repoGame Game
	if err := json.Unmarshal([]byte(gameJson), &repoGame); err != nil {
		return nil, err
	}

	return toDomainGame(&repoGame), nil
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

		var repoGame Game
		if err := json.Unmarshal([]byte(gameJson), &repoGame); err != nil {
			return nil, err
		}

		games = append(games, toDomainGame(&repoGame))
	}

	return games, nil
}

// SaveRating implements GameRepository.SaveRating
func (r *SQLiteRepository) SaveRating(ctx context.Context, gameID string, domainRating *domain.Rating) error {
	eventID, err := strconv.Atoi(gameID)
	if err != nil {
		return err
	}

	repoRating := toRepositoryRating(domainRating)

	_, err = r.db.ExecContext(ctx, `
		UPDATE results
		SET rating = ?,
			explanation = ?,
			spoiler_free_explanation = ?
		WHERE event_id = ?
	`,
		repoRating.Score,
		repoRating.Explanation,
		repoRating.SpoilerFree,
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

	var repoRating Rating
	err = r.db.QueryRowContext(ctx, `
		SELECT rating, explanation, spoiler_free_explanation
		FROM results
		WHERE event_id = ?
	`, eventID).Scan(
		&repoRating.Score,
		&repoRating.Explanation,
		&repoRating.SpoilerFree,
	)
	if err != nil {
		return nil, err
	}

	return toDomainRating(&repoRating), nil
}

// ListDates returns all unique date combinations from the games table
func (r *SQLiteRepository) ListDates(ctx context.Context) ([]domain.Date, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT season, week, season_type 
		FROM results 
		ORDER BY 
			season DESC,
			season_type DESC,
			week DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying dates: %w", err)
	}
	defer rows.Close()

	var dates []domain.Date
	for rows.Next() {
		var dbDate DBDate
		if err := rows.Scan(&dbDate.Season, &dbDate.Week, &dbDate.SeasonType); err != nil {
			return nil, fmt.Errorf("error scanning date: %w", err)
		}
		dates = append(dates, dbDate.ToDomainDate())
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dates: %w", err)
	}

	return dates, nil
}

// createTables creates the necessary database tables
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
