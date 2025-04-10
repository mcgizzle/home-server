package repository

import (
	"nfl/internal/domain"
	"strconv"
)

// DBDate represents a date record in the database
type DBDate struct {
	Season     int
	Week       int
	SeasonType int
}

// ToDBDate converts a domain.Date to a DBDate
func ToDBDate(d domain.Date) (DBDate, error) {
	season, err := strconv.Atoi(d.Season)
	if err != nil {
		return DBDate{}, err
	}

	week, err := strconv.Atoi(d.Week)
	if err != nil {
		return DBDate{}, err
	}

	var seasonType int
	switch d.SeasonType {
	case "preseason":
		seasonType = 1
	case "regular":
		seasonType = 2
	case "playoffs":
		seasonType = 3
	default:
		seasonType = 2 // default to regular season
	}

	return DBDate{
		Season:     season,
		Week:       week,
		SeasonType: seasonType,
	}, nil
}

// ToDomainDate converts a DBDate to a domain.Date
func (d DBDate) ToDomainDate() domain.Date {
	var seasonType string
	switch d.SeasonType {
	case 1:
		seasonType = "preseason"
	case 2:
		seasonType = "regular"
	case 3:
		seasonType = "playoffs"
	default:
		seasonType = "regular"
	}

	return domain.Date{
		Season:     strconv.Itoa(d.Season),
		Week:       strconv.Itoa(d.Week),
		SeasonType: seasonType,
	}
}
