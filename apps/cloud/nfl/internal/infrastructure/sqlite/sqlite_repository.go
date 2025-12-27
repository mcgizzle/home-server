package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// SQLiteRepository implements repository interfaces using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// CompetitionRepository implementation

func (r *SQLiteRepository) SaveCompetition(comp domain.Competition) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save teams first
	for _, ct := range comp.Teams {
		err = r.saveTeamTx(tx, ct.Team)
		if err != nil {
			return fmt.Errorf("failed to save team %s: %w", ct.Team.Name, err)
		}
	}

	// Save competition
	err = r.saveCompetitionTx(tx, comp)
	if err != nil {
		return fmt.Errorf("failed to save competition: %w", err)
	}

	// Save competition teams
	for _, ct := range comp.Teams {
		err = r.saveCompetitionTeamTx(tx, comp.ID, ct)
		if err != nil {
			return fmt.Errorf("failed to save competition team: %w", err)
		}
	}

	// Save rating if present
	if comp.Rating != nil {
		err = r.saveRatingTx(tx, comp.ID, *comp.Rating)
		if err != nil {
			return fmt.Errorf("failed to save rating: %w", err)
		}
	}

	// Save details if present
	if comp.Details != nil {
		err = r.saveCompetitionDetailsTx(tx, comp.ID, *comp.Details)
		if err != nil {
			return fmt.Errorf("failed to save competition details: %w", err)
		}
	}

	return tx.Commit()
}

func (r *SQLiteRepository) FindByPeriod(season, period, periodType string, sport domain.Sport) ([]domain.Competition, error) {
	query := `
		SELECT c.id 
		FROM competitions c
		WHERE c.season = ? AND c.period = ? AND c.period_type = ? AND c.sport_id = ?
		ORDER BY c.created_at DESC`

	rows, err := r.db.Query(query, season, period, periodType, sport)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var competitions []domain.Competition
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		comp, err := r.loadCompetition(id)
		if err != nil {
			return nil, err
		}
		competitions = append(competitions, comp)
	}

	return competitions, nil
}

func (r *SQLiteRepository) GetCompetitionByID(id string) (*domain.Competition, error) {
	competition, err := r.loadCompetition(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load competition %s: %w", id, err)
	}
	return &competition, nil
}

func (r *SQLiteRepository) GetAvailablePeriods(sport domain.Sport) ([]domain.Date, error) {
	query := `
		SELECT DISTINCT season, period, period_type 
		FROM competitions 
		WHERE sport_id = ? 
		ORDER BY season ASC, 
		         CASE 
		             WHEN period_type = 'preseason' THEN 0 
		             WHEN period_type = 'regular' THEN 1 
		             WHEN period_type = 'playoff' THEN 2 
		             ELSE 3 
		         END,
		         CAST(period AS INTEGER) ASC`

	rows, err := r.db.Query(query, sport)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []domain.Date
	for rows.Next() {
		var date domain.Date
		if err := rows.Scan(&date.Season, &date.Period, &date.PeriodType); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}

	return dates, nil
}

// RatingRepository implementation

func (r *SQLiteRepository) SaveRating(compID string, rating domain.Rating) error {
	query := `
		INSERT OR REPLACE INTO ratings 
		(competition_id, type, score, explanation, spoiler_free, source, confidence, generated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, compID, rating.Type, rating.Score, rating.Explanation,
		rating.SpoilerFree, rating.Source, rating.Confidence, rating.GeneratedAt)

	return err
}

func (r *SQLiteRepository) LoadRating(compID string, ratingType domain.RatingType) (domain.Rating, error) {
	query := `
		SELECT type, score, explanation, spoiler_free, source, confidence, generated_at
		FROM ratings WHERE competition_id = ? AND type = ?`

	var rating domain.Rating
	err := r.db.QueryRow(query, compID, ratingType).Scan(
		&rating.Type, &rating.Score, &rating.Explanation, &rating.SpoilerFree,
		&rating.Source, &rating.Confidence, &rating.GeneratedAt,
	)

	return rating, err
}

// TeamRepository implementation

func (r *SQLiteRepository) SaveTeam(team domain.Team) error {
	query := `
		INSERT OR REPLACE INTO teams (id, name, sport_id, logo_url)
		VALUES (?, ?, ?, ?)`

	_, err := r.db.Exec(query, team.ID, team.Name, team.Sport, team.LogoURL)
	return err
}

// SportRepository implementation

func (r *SQLiteRepository) ListSports() ([]repository.SportInfo, error) {
	query := `SELECT id, name FROM sports ORDER BY name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sports []repository.SportInfo
	for rows.Next() {
		var sport repository.SportInfo
		if err := rows.Scan(&sport.ID, &sport.Name); err != nil {
			return nil, err
		}
		sports = append(sports, sport)
	}

	return sports, nil
}

func (r *SQLiteRepository) GetSport(sportID string) (repository.SportInfo, error) {
	query := `SELECT id, name FROM sports WHERE id = ?`

	var sport repository.SportInfo
	err := r.db.QueryRow(query, sportID).Scan(&sport.ID, &sport.Name)
	return sport, err
}

// Private helper methods

func (r *SQLiteRepository) loadCompetition(id string) (domain.Competition, error) {
	// Load basic competition data
	comp, err := r.loadBasicCompetition(id)
	if err != nil {
		return domain.Competition{}, err
	}

	// Load teams
	teams, err := r.loadCompetitionTeams(id)
	if err != nil {
		return domain.Competition{}, fmt.Errorf("failed to load teams: %w", err)
	}
	comp.Teams = teams

	// Load rating
	rating, err := r.loadCompetitionRating(id)
	if err == nil {
		comp.Rating = &rating
	}

	// Load details
	details, err := r.loadCompetitionDetails(id)
	if err == nil {
		comp.Details = &details
	}

	return comp, nil
}

func (r *SQLiteRepository) saveTeamTx(tx *sql.Tx, team domain.Team) error {
	query := `INSERT OR REPLACE INTO teams (id, name, sport_id, logo_url) VALUES (?, ?, ?, ?)`
	_, err := tx.Exec(query, team.ID, team.Name, team.Sport, team.LogoURL)
	return err
}

func (r *SQLiteRepository) saveCompetitionTx(tx *sql.Tx, comp domain.Competition) error {
	query := `
		INSERT OR REPLACE INTO competitions 
		(id, event_id, sport_id, season, period, period_type, start_time, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := tx.Exec(query, comp.ID, comp.EventID, comp.Sport, comp.Season,
		comp.Period, comp.PeriodType, comp.StartTime, comp.Status, comp.CreatedAt)
	return err
}

func (r *SQLiteRepository) saveCompetitionTeamTx(tx *sql.Tx, compID string, ct domain.CompetitionTeam) error {
	statsJSON, err := json.Marshal(ct.Stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO competition_teams 
		(competition_id, team_id, home_away, score, stats)
		VALUES (?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, compID, ct.Team.ID, ct.HomeAway, ct.Score, string(statsJSON))
	return err
}

func (r *SQLiteRepository) saveRatingTx(tx *sql.Tx, compID string, rating domain.Rating) error {
	query := `
		INSERT OR REPLACE INTO ratings 
		(competition_id, type, score, explanation, spoiler_free, source, confidence, generated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := tx.Exec(query, compID, rating.Type, rating.Score, rating.Explanation,
		rating.SpoilerFree, rating.Source, rating.Confidence, rating.GeneratedAt)
	return err
}

func (r *SQLiteRepository) saveCompetitionDetailsTx(tx *sql.Tx, compID string, details domain.CompetitionDetails) error {
	playByPlayJSON, err := json.Marshal(details.PlayByPlay)
	if err != nil {
		return fmt.Errorf("failed to marshal play by play: %w", err)
	}

	metadataJSON, err := json.Marshal(details.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO competition_details 
		(competition_id, play_by_play, metadata)
		VALUES (?, ?, ?)`

	_, err = tx.Exec(query, compID, string(playByPlayJSON), string(metadataJSON))
	return err
}

func (r *SQLiteRepository) loadBasicCompetition(id string) (domain.Competition, error) {
	query := `
		SELECT id, event_id, sport_id, season, period, period_type, start_time, status, created_at
		FROM competitions WHERE id = ?`

	var comp domain.Competition
	var startTime sql.NullTime
	err := r.db.QueryRow(query, id).Scan(
		&comp.ID, &comp.EventID, &comp.Sport, &comp.Season, &comp.Period,
		&comp.PeriodType, &startTime, &comp.Status, &comp.CreatedAt,
	)
	if err != nil {
		return domain.Competition{}, err
	}

	if startTime.Valid {
		comp.StartTime = &startTime.Time
	}

	return comp, nil
}

func (r *SQLiteRepository) loadCompetitionTeams(compID string) ([]domain.CompetitionTeam, error) {
	query := `
		SELECT t.id, t.name, t.sport_id, t.logo_url, ct.home_away, ct.score, ct.stats
		FROM competition_teams ct
		JOIN teams t ON ct.team_id = t.id
		WHERE ct.competition_id = ?`

	rows, err := r.db.Query(query, compID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []domain.CompetitionTeam
	for rows.Next() {
		var team domain.Team
		var ct domain.CompetitionTeam
		var statsJSON string

		err := rows.Scan(&team.ID, &team.Name, &team.Sport, &team.LogoURL,
			&ct.HomeAway, &ct.Score, &statsJSON)
		if err != nil {
			return nil, err
		}

		// Unmarshal stats
		if statsJSON != "" {
			err = json.Unmarshal([]byte(statsJSON), &ct.Stats)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
			}
		}

		ct.Team = team
		teams = append(teams, ct)
	}

	return teams, nil
}

func (r *SQLiteRepository) loadCompetitionRating(compID string) (domain.Rating, error) {
	query := `
		SELECT type, score, explanation, spoiler_free, source, confidence, generated_at
		FROM ratings WHERE competition_id = ? LIMIT 1`

	var rating domain.Rating
	err := r.db.QueryRow(query, compID).Scan(
		&rating.Type, &rating.Score, &rating.Explanation, &rating.SpoilerFree,
		&rating.Source, &rating.Confidence, &rating.GeneratedAt,
	)

	return rating, err
}

func (r *SQLiteRepository) loadCompetitionDetails(compID string) (domain.CompetitionDetails, error) {
	query := `SELECT play_by_play, metadata FROM competition_details WHERE competition_id = ?`

	var details domain.CompetitionDetails
	var playByPlayJSON, metadataJSON string

	err := r.db.QueryRow(query, compID).Scan(&playByPlayJSON, &metadataJSON)
	if err != nil {
		return domain.CompetitionDetails{}, err
	}

	// Unmarshal JSON fields
	if playByPlayJSON != "" {
		err = json.Unmarshal([]byte(playByPlayJSON), &details.PlayByPlay)
		if err != nil {
			return domain.CompetitionDetails{}, fmt.Errorf("failed to unmarshal play by play: %w", err)
		}
	}

	if metadataJSON != "" {
		err = json.Unmarshal([]byte(metadataJSON), &details.Metadata)
		if err != nil {
			return domain.CompetitionDetails{}, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return details, nil
}

// SentimentRatingRepository implementation

func (r *SQLiteRepository) SaveSentimentRating(rating domain.SentimentRating, competitionID string) error {
	// Generate a unique ID for the sentiment rating
	id := fmt.Sprintf("%s_%s", competitionID, rating.Source)

	// Marshal highlights to JSON
	highlightsJSON, err := json.Marshal(rating.Highlights)
	if err != nil {
		return fmt.Errorf("failed to marshal highlights: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO sentiment_ratings
		(id, competition_id, source, thread_url, comment_count, score, sentiment, highlights, generated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.Exec(query, id, competitionID, rating.Source, rating.ThreadURL,
		rating.CommentCount, rating.Score, rating.Sentiment, string(highlightsJSON), rating.GeneratedAt)

	return err
}

func (r *SQLiteRepository) GetSentimentRating(competitionID string) (*domain.SentimentRating, error) {
	query := `
		SELECT source, thread_url, comment_count, score, sentiment, highlights, generated_at
		FROM sentiment_ratings WHERE competition_id = ? LIMIT 1`

	var rating domain.SentimentRating
	var highlightsJSON string

	err := r.db.QueryRow(query, competitionID).Scan(
		&rating.Source, &rating.ThreadURL, &rating.CommentCount, &rating.Score,
		&rating.Sentiment, &highlightsJSON, &rating.GeneratedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal highlights JSON
	if highlightsJSON != "" {
		err = json.Unmarshal([]byte(highlightsJSON), &rating.Highlights)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal highlights: %w", err)
		}
	}

	return &rating, nil
}
