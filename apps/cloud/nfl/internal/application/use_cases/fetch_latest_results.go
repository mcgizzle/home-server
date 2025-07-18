package use_cases

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/application"
	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// FetchLatestResultsUseCase defines the business operation for fetching latest NFL results
type FetchLatestResultsUseCase interface {
	Execute() ([]domain.Result, error)
}

// fetchLatestResultsUseCase implements FetchLatestResultsUseCase
type fetchLatestResultsUseCase struct {
	espnClient external.ESPNClient
	resultRepo repository.ResultRepository
	ratingSvc  application.RatingService
}

// NewFetchLatestResultsUseCase creates a new instance of FetchLatestResultsUseCase
func NewFetchLatestResultsUseCase(espnClient external.ESPNClient, resultRepo repository.ResultRepository, ratingSvc application.RatingService) FetchLatestResultsUseCase {
	return &fetchLatestResultsUseCase{
		espnClient: espnClient,
		resultRepo: resultRepo,
		ratingSvc:  ratingSvc,
	}
}

// Execute fetches the latest NFL results from ESPN and processes them
func (uc *fetchLatestResultsUseCase) Execute() ([]domain.Result, error) {
	eventRefs, err := uc.espnClient.ListLatestEvents()
	if err != nil {
		log.Printf("Error listing latest events: %v", err)
		return []domain.Result{}, err
	}

	season := eventRefs.Meta.Parameters.Season[0]
	week := eventRefs.Meta.Parameters.Week[0]
	seasonType := eventRefs.Meta.Parameters.SeasonTypes[0]

	// Load existing results to avoid duplicates
	existingResults, err := uc.resultRepo.LoadResults(season, week, seasonType)
	if err != nil {
		log.Printf("Error loading existing results: %v", err)
		return []domain.Result{}, err
	}

	// Filter out events that have already been processed
	var filteredEventRefs []external.EventRef
	for _, eventRef := range eventRefs.Items {
		event, err := uc.espnClient.GetEvent(eventRef.Ref)
		if err != nil {
			log.Printf("Error getting event: %v", err)
			continue
		}
		shouldInclude := true
		for _, result := range existingResults {
			if result.EventId == event.Id {
				shouldInclude = false
				log.Printf("Event already processed: %s - %s", season, week)
				break
			}
		}
		if shouldInclude {
			filteredEventRefs = append(filteredEventRefs, eventRef)
		}
	}

	var results []domain.Result
	for _, eventRef := range filteredEventRefs {
		log.Printf("Processing event: Season %s - Week %s - Season Type %s", season, week, seasonType)
		event, err := uc.espnClient.GetEvent(eventRef.Ref)
		if err != nil {
			log.Printf("Error getting event: %v", err)
			continue
		}
		maybeGame := uc.espnClient.GetTeamAndScore(event)

		// Game has not been played yet
		if maybeGame == nil {
			log.Printf("Game has not been played yet, skipping")
			continue
		}
		game := *maybeGame

		rating := uc.ratingSvc.ProduceRating(game)

		result := domain.Result{
			EventId:    event.Id,
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
