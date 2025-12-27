package repository

import (
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
)

// SentimentRatingRepository handles sentiment rating operations
type SentimentRatingRepository interface {
	SaveSentimentRating(rating domain.SentimentRating, competitionID string) error
	GetSentimentRating(competitionID string) (*domain.SentimentRating, error)
}
