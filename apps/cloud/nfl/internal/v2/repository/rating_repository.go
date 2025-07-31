package repository

import (
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
)

// RatingRepository handles rating-specific operations
type RatingRepository interface {
	SaveRating(compID string, rating domain.Rating) error
	LoadRating(compID string, ratingType domain.RatingType) (domain.Rating, error)
}
