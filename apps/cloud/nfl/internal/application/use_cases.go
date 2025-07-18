package application

import (
	"log"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// UseCase interfaces define the business operations
type FetchLatestResultsUseCase interface {
	Execute() ([]domain.Result, error)
}

type FetchSpecificResultsUseCase interface {
	Execute(season, week, seasonType string) ([]domain.Result, error)
}

type GetAvailableDatesUseCase interface {
	Execute() ([]domain.Date, error)
}

type SaveResultsUseCase interface {
	Execute(results []domain.Result) error
}

type GetTemplateDataUseCase interface {
	Execute(season, week, seasonType string) (domain.TemplateData, error)
}

// Use case implementations
type fetchLatestResultsUseCase struct {
	espnClient external.ESPNClient
	resultRepo repository.ResultRepository
	ratingSvc  RatingService
}

type fetchSpecificResultsUseCase struct {
	espnClient external.ESPNClient
	resultRepo repository.ResultRepository
	ratingSvc  RatingService
}

type getAvailableDatesUseCase struct {
	resultRepo repository.ResultRepository
}

type saveResultsUseCase struct {
	resultRepo repository.ResultRepository
}

type getTemplateDataUseCase struct {
	resultRepo repository.ResultRepository
}

// New use case constructors
func NewFetchLatestResultsUseCase(espnClient external.ESPNClient, resultRepo repository.ResultRepository, ratingSvc RatingService) FetchLatestResultsUseCase {
	return &fetchLatestResultsUseCase{
		espnClient: espnClient,
		resultRepo: resultRepo,
		ratingSvc:  ratingSvc,
	}
}

func NewFetchSpecificResultsUseCase(espnClient external.ESPNClient, resultRepo repository.ResultRepository, ratingSvc RatingService) FetchSpecificResultsUseCase {
	return &fetchSpecificResultsUseCase{
		espnClient: espnClient,
		resultRepo: resultRepo,
		ratingSvc:  ratingSvc,
	}
}

func NewGetAvailableDatesUseCase(resultRepo repository.ResultRepository) GetAvailableDatesUseCase {
	return &getAvailableDatesUseCase{
		resultRepo: resultRepo,
	}
}

func NewSaveResultsUseCase(resultRepo repository.ResultRepository) SaveResultsUseCase {
	return &saveResultsUseCase{
		resultRepo: resultRepo,
	}
}

func NewGetTemplateDataUseCase(resultRepo repository.ResultRepository) GetTemplateDataUseCase {
	return &getTemplateDataUseCase{
		resultRepo: resultRepo,
	}
}

// Implementation of fetchLatestResultsUseCase
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

// Implementation of fetchSpecificResultsUseCase
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

// Implementation of getAvailableDatesUseCase
func (uc *getAvailableDatesUseCase) Execute() ([]domain.Date, error) {
	return uc.resultRepo.LoadDates()
}

// Implementation of saveResultsUseCase
func (uc *saveResultsUseCase) Execute(results []domain.Result) error {
	return uc.resultRepo.SaveResults(results)
}

// Implementation of getTemplateDataUseCase
func (uc *getTemplateDataUseCase) Execute(season, week, seasonType string) (domain.TemplateData, error) {
	var results []domain.Result
	var err error

	if season != "" && week != "" && seasonType != "" {
		seasonTypeNumber := domain.SeasonTypeToNumber(seasonType)
		results, err = uc.resultRepo.LoadResults(season, week, seasonTypeNumber)
		if err != nil {
			return domain.TemplateData{}, err
		}
	} else {
		// Use the most recent week with results
		dates, err := uc.resultRepo.LoadDates()
		if err != nil {
			return domain.TemplateData{}, err
		}
		if len(dates) > 0 {
			mostRecentDate := dates[0]
			week = mostRecentDate.Week
			season = mostRecentDate.Season
			seasonType = mostRecentDate.SeasonType
			results, err = uc.resultRepo.LoadResults(season, week, seasonType)
			if err != nil {
				return domain.TemplateData{}, err
			}
		} else {
			// Database is empty - return empty state
			return domain.TemplateData{
				Results: []domain.Result{},
				Dates:   []domain.DateTemplate{},
				Current: domain.DateTemplate{
					Season:             "No data",
					Week:               "available",
					SeasonTypeShowable: "yet",
					SeasonType:         "",
				},
			}, nil
		}
	}

	dates, err := uc.resultRepo.LoadDates()
	if err != nil {
		return domain.TemplateData{}, err
	}

	dateTemplates := make([]domain.DateTemplate, len(dates))
	for i, date := range dates {
		dateTemplates[i] = date.Template()
	}

	return domain.TemplateData{
		Results: results,
		Dates:   dateTemplates,
		Current: domain.Date{
			Season:     season,
			Week:       week,
			SeasonType: seasonType,
		}.Template(),
	}, nil
}
