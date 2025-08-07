package services

import "github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"

// RatingService defines the interface for generating ratings for competitions.
// This interface is owned by the application layer and implemented by infrastructure.
// It follows clean architecture principles by defining what the business logic needs,
// not how the external rating provider (OpenAI, Claude, etc.) structures its responses.
type RatingService interface {
	// ProduceRatingForCompetition generates a rating for a competition
	// Returns a domain.Rating with score, explanations, and metadata
	ProduceRatingForCompetition(comp domain.Competition) (domain.Rating, error)
}
