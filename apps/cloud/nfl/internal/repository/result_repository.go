package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// ResultRepository defines the interface for database operations on results
type ResultRepository interface {
	SaveResults(results []domain.Result) error
	LoadResults(season, week, seasonType string) ([]domain.Result, error)
	LoadDates() ([]domain.Date, error)
}

// SQLiteResultRepository implements ResultRepository using SQLite
type SQLiteResultRepository struct {
	db *sql.DB
}

// NewSQLiteResultRepository creates a new SQLite repository instance
func NewSQLiteResultRepository(db *sql.DB) *SQLiteResultRepository {
	return &SQLiteResultRepository{db: db}
}

// SaveResults saves multiple results to the database
func (r *SQLiteResultRepository) SaveResults(results []domain.Result) error {
	stmt, err := r.db.Prepare("insert into results(event_id, season, week, season_type, rating, explanation, spoiler_free_explanation, game) values(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(stmt)

	for _, result := range results {
		gameJson, err := json.Marshal(result.Game)
		if err != nil {
			return err
		}

		weekAsInt, err := strconv.Atoi(result.Week)
		if err != nil {
			return err
		}
		seasonAsInt, err := strconv.Atoi(result.Season)
		if err != nil {
			return err
		}

		seasonTypeAsInt, err := strconv.Atoi(result.SeasonType)
		if err != nil {
			return err
		}

		eventIdAsInt, err := strconv.Atoi(result.EventId)
		if err != nil {
			return err
		}

		_, err = stmt.Exec(eventIdAsInt, seasonAsInt, weekAsInt, seasonTypeAsInt, result.Rating.Score, result.Rating.Explanation, result.Rating.SpoilerFree, string(gameJson))
		if err != nil {
			return err
		}
	}
	log.Printf("Saved %d results", len(results))
	return nil
}

// LoadResults loads results from the database for a specific season, week, and season type
func (r *SQLiteResultRepository) LoadResults(season, week, seasonType string) ([]domain.Result, error) {
	if season == "" || week == "" || seasonType == "" {
		return nil, fmt.Errorf("Season or week or season type not provided")
	}

	selectQuery := "select id, event_id, season, week, season_type, rating, explanation, spoiler_free_explanation, game from results where season = ? and week = ? and season_type = ? order by season desc, season_type desc, week desc, rating desc"

	rows, err := r.db.Query(selectQuery, season, week, seasonType)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	var results []domain.Result

	for rows.Next() {
		var result domain.Result
		var gameJson string
		err = rows.Scan(&result.Id, &result.EventId, &result.Season, &result.Week, &result.SeasonType, &result.Rating.Score, &result.Rating.Explanation, &result.Rating.SpoilerFree, &gameJson)

		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(gameJson), &result.Game)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

// LoadDates loads all distinct dates from the database
func (r *SQLiteResultRepository) LoadDates() ([]domain.Date, error) {
	selectQuery := "select distinct season, week, season_type from results order by season_type desc, season desc, week desc"

	rows, err := r.db.Query(selectQuery)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(rows)

	var dates []domain.Date

	for rows.Next() {
		var season string
		var week string
		var seasonType string
		err = rows.Scan(&season, &week, &seasonType)

		if err != nil {
			return nil, err
		}

		dates = append(dates, domain.Date{
			Season:     season,
			Week:       week,
			SeasonType: seasonType,
		})
	}

	return dates, nil
}
