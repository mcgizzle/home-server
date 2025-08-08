package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application/services"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// GenerateRatingsUseCase defines the V2 business operation for generating missing ratings
type GenerateRatingsUseCase interface {
	Execute(sportID string) (int, error) // returns number of ratings generated
}

// generateRatingsUseCase implements GenerateRatingsUseCase
type generateRatingsUseCase struct {
	competitionRepo repository.CompetitionRepository
	ratingRepo      repository.RatingRepository
	ratingService   services.RatingService
}

// NewGenerateRatingsUseCase creates a new instance of GenerateRatingsUseCase
func NewGenerateRatingsUseCase(
	competitionRepo repository.CompetitionRepository,
	ratingRepo repository.RatingRepository,
	ratingService services.RatingService,
) GenerateRatingsUseCase {
	return &generateRatingsUseCase{
		competitionRepo: competitionRepo,
		ratingRepo:      ratingRepo,
		ratingService:   ratingService,
	}
}

// Execute finds competitions without ratings and generates missing ratings
func (uc *generateRatingsUseCase) Execute(sportID string) (int, error) {
	sport := domain.Sport(sportID)
	ratingsGenerated := 0

	// Get all available periods for the sport
	dates, err := uc.competitionRepo.GetAvailablePeriods(sport)
	if err != nil {
		log.Printf("Error loading available periods for sport %s: %v", sportID, err)
		return 0, err
	}

	// Process each period to find competitions without ratings
	for _, date := range dates {
		competitions, err := uc.competitionRepo.FindByPeriod(date.Season, date.Period, date.PeriodType, sport)
		if err != nil {
			log.Printf("Error loading competitions for %s %s %s: %v", date.Season, date.Period, date.PeriodType, err)
			continue
		}

		// Check each competition for missing ratings
		for _, competition := range competitions {
			// Check if this competition already has an excitement rating
			_, err := uc.ratingRepo.LoadRating(competition.ID, domain.RatingTypeExcitement)
			if err != nil {
				// Rating doesn't exist, generate one
				log.Printf("Generating missing rating for competition %s", competition.ID)

				rating, err := uc.ratingService.ProduceRatingForCompetition(competition)
				if err != nil {
					log.Printf("Error generating rating for competition %s: %v", competition.ID, err)
					continue
				}

				// Save the new rating
				err = uc.ratingRepo.SaveRating(competition.ID, rating)
				if err != nil {
					log.Printf("Error saving rating for competition %s: %v", competition.ID, err)
					continue
				}

				ratingsGenerated++
				log.Printf("Successfully generated rating for competition %s (score: %d)", competition.ID, rating.Score)
			}
		}
	}

	if ratingsGenerated > 0 {
		log.Printf("Generated %d missing ratings for sport %s", ratingsGenerated, sportID)
	} else {
		log.Printf("No missing ratings found for sport %s", sportID)
	}

	return ratingsGenerated, nil
}
