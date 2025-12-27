package use_cases

import (
	"log"
	"slices"
	"sort"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// GetTemplateDataUseCase defines the V2 business operation for getting template data
type GetTemplateDataUseCase interface {
	Execute(sportID, season, period, periodType string) (domain.TemplateData, error)
}

// getTemplateDataUseCase implements GetTemplateDataUseCase
type getTemplateDataUseCase struct {
	competitionRepo repository.CompetitionRepository
	sentimentRepo   repository.SentimentRatingRepository
}

// NewGetTemplateDataUseCase creates a new instance of V2 GetTemplateDataUseCase
func NewGetTemplateDataUseCase(competitionRepo repository.CompetitionRepository, sentimentRepo repository.SentimentRatingRepository) GetTemplateDataUseCase {
	return &getTemplateDataUseCase{
		competitionRepo: competitionRepo,
		sentimentRepo:   sentimentRepo,
	}
}

// Execute gets template data for a specific sport and period
func (uc *getTemplateDataUseCase) Execute(sportID, season, period, periodType string) (domain.TemplateData, error) {
	sport := domain.Sport(sportID)

	// Load competitions for the specified parameters
	competitions, err := uc.competitionRepo.FindByPeriod(season, period, periodType, sport)
	if err != nil {
		log.Printf("Error loading competitions: %v", err)
		return domain.TemplateData{}, err
	}

	// Load all available dates for this sport
	dates, err := uc.competitionRepo.GetAvailablePeriods(sport)
	if err != nil {
		log.Printf("Error loading dates: %v", err)
		return domain.TemplateData{}, err
	}

	// Convert competitions to template format with computed categories
	var templateResults []domain.TemplateResult
	for _, competition := range competitions {
		result := competition.ToTemplateResult()

		// Fetch sentiment rating if available
		if sentimentRating, err := uc.sentimentRepo.GetSentimentRating(competition.ID); err == nil && sentimentRating != nil {
			result.Sentiment = domain.TemplateSentiment{
				Score:        sentimentRating.Score,
				Sentiment:    sentimentRating.Sentiment,
				Highlights:   sentimentRating.Highlights,
				CommentCount: sentimentRating.CommentCount,
				ThreadURL:    sentimentRating.ThreadURL,
				HasData:      true,
			}
		}

		templateResults = append(templateResults, result)
	}

	// Sort template results by rating score (highest first), with unrated games at the end
	sort.Slice(templateResults, func(i, j int) bool {
		ratingI := templateResults[i].Rating.Score
		ratingJ := templateResults[j].Rating.Score

		// If both are unrated (score 0), maintain original order
		if ratingI == 0 && ratingJ == 0 {
			return false
		}
		// If one is unrated, put it after the rated one
		if ratingI == 0 {
			return false
		}
		if ratingJ == 0 {
			return true
		}
		// Both are rated, sort by score descending
		return ratingI > ratingJ
	})

	// Convert dates to template format
	var templateDates []domain.DateTemplate
	for _, date := range dates {
		templateDates = append(templateDates, date.Template())
	}

	// Create current date template
	currentDate := domain.Date{
		Season:     season,
		Period:     period,
		PeriodType: periodType,
	}

	// Extract unique seasons
	seasons := []string{}
	for _, date := range dates {
		if !slices.Contains(seasons, date.Season) {
			seasons = append(seasons, date.Season)
		}
	}

	templateData := domain.TemplateData{
		Results: templateResults,
		Dates:   templateDates,
		Seasons: seasons,
		Current: currentDate.Template(),
	}

	log.Printf("Successfully retrieved template data with %d competitions and %d dates for sport %s", len(competitions), len(templateDates), sportID)
	return templateData, nil
}
