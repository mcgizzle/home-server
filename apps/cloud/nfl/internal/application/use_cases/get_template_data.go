package use_cases

import (
	"log"
	"slices"

	"github.com/mcgizzle/home-server/apps/cloud/internal/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// GetTemplateDataUseCase defines the business operation for getting template data
type GetTemplateDataUseCase interface {
	Execute(season, week, seasonType string) (domain.TemplateData, error)
}

// getTemplateDataUseCase implements GetTemplateDataUseCase
type getTemplateDataUseCase struct {
	resultRepo repository.ResultRepository
}

// NewGetTemplateDataUseCase creates a new instance of GetTemplateDataUseCase
func NewGetTemplateDataUseCase(resultRepo repository.ResultRepository) GetTemplateDataUseCase {
	return &getTemplateDataUseCase{
		resultRepo: resultRepo,
	}
}

// Execute gets template data for the UI
func (uc *getTemplateDataUseCase) Execute(season, week, seasonType string) (domain.TemplateData, error) {
	// Load results for the specified parameters
	results, err := uc.resultRepo.LoadResults(season, week, seasonType)
	if err != nil {
		log.Printf("Error loading results: %v", err)
		return domain.TemplateData{}, err
	}

	// Load all available dates
	dates, err := uc.resultRepo.LoadDates()
	if err != nil {
		log.Printf("Error loading dates: %v", err)
		return domain.TemplateData{}, err
	}

	// Convert results to template format with computed categories
	var templateResults []domain.TemplateResult
	for _, result := range results {
		templateResults = append(templateResults, result.ToTemplateResult())
	}

	// Convert dates to template format
	var templateDates []domain.DateTemplate
	for _, date := range dates {
		templateDates = append(templateDates, date.Template())
	}

	// Create current date template
	currentDate := domain.Date{
		Season:     season,
		Week:       week,
		SeasonType: seasonType,
	}

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

	log.Printf("Successfully retrieved template data with %d results", len(results))
	return templateData, nil
}
