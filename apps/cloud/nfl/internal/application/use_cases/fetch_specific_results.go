package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// FetchSpecificResultsUseCase defines the business operation for fetching specific NFL results
type FetchSpecificResultsUseCase interface {
	Execute(season, week, seasonType string) ([]domain.Result, error)
}

// fetchSpecificResultsUseCase implements FetchSpecificResultsUseCase
type fetchSpecificResultsUseCase struct {
	espnClient external.ESPNClient
	resultRepo repository.ResultRepository
	ratingSvc  application.RatingService
}

// NewFetchSpecificResultsUseCase creates a new instance of FetchSpecificResultsUseCase
func NewFetchSpecificResultsUseCase(espnClient external.ESPNClient, resultRepo repository.ResultRepository, ratingSvc application.RatingService) FetchSpecificResultsUseCase {
	return &fetchSpecificResultsUseCase{
		espnClient: espnClient,
		resultRepo: resultRepo,
		ratingSvc:  ratingSvc,
	}
}

// Execute fetches specific NFL results from ESPN and processes them
func (uc *fetchSpecificResultsUseCase) Execute(season, week, seasonType string) ([]domain.Result, error) {
	specificEvents, err := uc.espnClient.ListSpecificEvents(season, week, seasonType)
	if err != nil {
		log.Printf("Error listing specific events: %v", err)
		return []domain.Result{}, err
	}

	var results []domain.Result
	for _, eventId := range specificEvents.Events {
		log.Printf("Processing event: %s - %s", season, week)
		event, err := uc.espnClient.GetEventById(eventId.Id)
		if err != nil {
			log.Printf("Error getting event by ID: %v", err)
			continue
		}
		maybeGame := uc.espnClient.GetTeamAndScore(event)
		if maybeGame == nil {
			log.Printf("Game has not been played yet, skipping")
			continue
		}
		game := *maybeGame

		rating := uc.ratingSvc.ProduceRating(game)

		result := domain.Result{
			EventId:    eventId.Id,
			Season:     season,
			SeasonType: seasonType,
			Week:       week,
			Rating:     rating,
			Game:       game,
		}
		results = append(results, result)
	}

	log.Printf("Produced %d results", len(results))
	return results, nil
}
