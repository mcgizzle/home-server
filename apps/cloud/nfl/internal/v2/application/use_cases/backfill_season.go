package use_cases

import (
	"fmt"
	"log"
	"strconv"

	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository"
)

// BackfillSeasonUseCase defines the V2 business operation for backfilling missing competition data
type BackfillSeasonUseCase interface {
	Execute(sportID, season string) (*BackfillResult, error)
	ExecuteWithLimit(sportID, season string, limit int) (*BackfillResult, error)
}

// BackfillResult represents the result of a backfill operation
type BackfillResult struct {
	Season            string                 `json:"season"`
	Limit             int                    `json:"limit,omitempty"`         // Competition limit (0 = no limit)
	LimitReached      bool                   `json:"limit_reached,omitempty"` // Whether the limit was reached
	PeriodsProcessed  int                    `json:"periods_processed"`
	CompetitionsAdded int                    `json:"competitions_added"`
	Errors            []BackfillError        `json:"errors,omitempty"`
	PeriodResults     []BackfillPeriodResult `json:"period_results"`
}

// BackfillPeriodResult represents the result for a specific period
type BackfillPeriodResult struct {
	Period        string `json:"period"`
	PeriodType    string `json:"period_type"`
	ExistingCount int    `json:"existing_count"`
	FetchedCount  int    `json:"fetched_count"`
	AddedCount    int    `json:"added_count"`
	Skipped       bool   `json:"skipped"`
	SkipReason    string `json:"skip_reason,omitempty"`
	Error         string `json:"error,omitempty"`
}

// BackfillError represents an error that occurred during backfill
type BackfillError struct {
	Period     string `json:"period"`
	PeriodType string `json:"period_type"`
	Error      string `json:"error"`
}

// backfillSeasonUseCase implements BackfillSeasonUseCase
type backfillSeasonUseCase struct {
	competitionRepo      repository.CompetitionRepository
	fetchSpecificUseCase FetchSpecificCompetitionsUseCase
	saveUseCase          SaveCompetitionsUseCase
}

// NewBackfillSeasonUseCase creates a new instance of BackfillSeasonUseCase
func NewBackfillSeasonUseCase(
	competitionRepo repository.CompetitionRepository,
	fetchSpecificUseCase FetchSpecificCompetitionsUseCase,
	saveUseCase SaveCompetitionsUseCase,
) BackfillSeasonUseCase {
	return &backfillSeasonUseCase{
		competitionRepo:      competitionRepo,
		fetchSpecificUseCase: fetchSpecificUseCase,
		saveUseCase:          saveUseCase,
	}
}

// Execute performs backfill for all periods in a given season
func (uc *backfillSeasonUseCase) Execute(sportID, season string) (*BackfillResult, error) {
	return uc.ExecuteWithLimit(sportID, season, -1) // -1 means no limit
}

// ExecuteWithLimit performs backfill for periods in a given season with a competition limit
func (uc *backfillSeasonUseCase) ExecuteWithLimit(sportID, season string, limit int) (*BackfillResult, error) {
	sport := domain.Sport(sportID)

	limitMsg := ""
	if limit > 0 {
		limitMsg = fmt.Sprintf(" (limit: %d competitions)", limit)
	}
	log.Printf("Starting backfill for %s season %s%s", sportID, season, limitMsg)

	result := &BackfillResult{
		Season:        season,
		Limit:         limit,
		PeriodResults: []BackfillPeriodResult{},
		Errors:        []BackfillError{},
	}

	// Get periods to backfill based on sport
	periods := uc.getPeriodsForSeason(sportID, season)

	for _, periodInfo := range periods {
		// Check if we've reached the limit
		if limit > 0 && result.CompetitionsAdded >= limit {
			log.Printf("Reached competition limit of %d, stopping backfill", limit)
			result.LimitReached = true
			break
		}

		// Calculate remaining limit for this period
		remainingLimit := -1 // No limit
		if limit > 0 {
			remainingLimit = limit - result.CompetitionsAdded
			if remainingLimit <= 0 {
				break
			}
		}

		periodResult := uc.backfillPeriodWithLimit(sport, season, periodInfo.Period, periodInfo.PeriodType, remainingLimit)
		result.PeriodResults = append(result.PeriodResults, periodResult)

		if periodResult.Error != "" {
			result.Errors = append(result.Errors, BackfillError{
				Period:     periodInfo.Period,
				PeriodType: periodInfo.PeriodType,
				Error:      periodResult.Error,
			})
		}

		result.PeriodsProcessed++
		result.CompetitionsAdded += periodResult.AddedCount

		// Check if we've reached the limit after processing this period
		if limit > 0 && result.CompetitionsAdded >= limit {
			log.Printf("Reached competition limit of %d after processing %s %s",
				limit, periodInfo.Period, periodInfo.PeriodType)
			result.LimitReached = true
			break
		}
	}

	log.Printf("Backfill completed for %s season %s: %d periods processed, %d competitions added, %d errors%s",
		sportID, season, result.PeriodsProcessed, result.CompetitionsAdded, len(result.Errors), limitMsg)

	return result, nil
}

// backfillPeriodWithLimit handles backfill for a specific period with a competition limit
func (uc *backfillSeasonUseCase) backfillPeriodWithLimit(sport domain.Sport, season, period, periodType string, remainingLimit int) BackfillPeriodResult {
	result := BackfillPeriodResult{
		Period:     period,
		PeriodType: periodType,
	}

	// Check existing competitions for this period
	existingCompetitions, err := uc.competitionRepo.FindByPeriod(season, period, periodType, sport)
	if err != nil {
		result.Error = fmt.Sprintf("Error checking existing competitions: %v", err)
		return result
	}

	result.ExistingCount = len(existingCompetitions)

	// If we already have competitions for this period, check if we should skip or still fetch
	if len(existingCompetitions) > 0 {
		log.Printf("Found %d existing competitions for %s %s %s - still fetching to check for updates",
			len(existingCompetitions), season, period, periodType)
	}

	// Fetch competitions for this period with limit
	var fetchedCompetitions []domain.Competition
	if remainingLimit > 0 {
		fetchedCompetitions, err = uc.fetchSpecificUseCase.ExecuteWithLimit(string(sport), season, period, periodType, remainingLimit)
	} else {
		fetchedCompetitions, err = uc.fetchSpecificUseCase.Execute(string(sport), season, period, periodType)
	}

	if err != nil {
		result.Error = fmt.Sprintf("Error fetching competitions: %v", err)
		return result
	}

	result.FetchedCount = len(fetchedCompetitions)

	if len(fetchedCompetitions) == 0 {
		result.Skipped = true
		result.SkipReason = "No competitions found from ESPN for this period"
		log.Printf("No competitions found for %s %s %s", season, period, periodType)
		return result
	}

	// Since we already applied the limit in the fetch, no need to limit again
	competitionsToSave := fetchedCompetitions

	// Save the competitions (the fetch use case already handles deduplication)
	if len(competitionsToSave) > 0 {
		err = uc.saveUseCase.Execute(competitionsToSave)
		if err != nil {
			result.Error = fmt.Sprintf("Error saving competitions: %v", err)
			return result
		}
	}

	result.AddedCount = len(competitionsToSave)

	if remainingLimit > 0 {
		log.Printf("Backfilled %s %s %s: %d competitions saved (with limit %d)",
			season, period, periodType, result.AddedCount, remainingLimit)
	} else {
		log.Printf("Backfilled %s %s %s: %d competitions saved",
			season, period, periodType, result.AddedCount)
	}

	return result
}

// PeriodInfo represents a period to be backfilled
type PeriodInfo struct {
	Period     string
	PeriodType string
}

// getPeriodsForSeason returns the periods that should be checked for backfill
func (uc *backfillSeasonUseCase) getPeriodsForSeason(sportID, season string) []PeriodInfo {
	switch sportID {
	case "nfl":
		return uc.getNFLPeriods(season)
	default:
		log.Printf("Unknown sport %s for backfill, using default NFL periods", sportID)
		return uc.getNFLPeriods(season)
	}
}

// getNFLPeriods returns all possible NFL periods for a season
// TDOD: get this data from the ESPN API
func (uc *backfillSeasonUseCase) getNFLPeriods(season string) []PeriodInfo {
	var periods []PeriodInfo

	// Regular season weeks (1-18 for modern NFL)
	for week := 1; week <= 18; week++ {
		periods = append(periods, PeriodInfo{
			Period:     strconv.Itoa(week),
			PeriodType: "regular", // Use lowercase to match existing main.go usage
		})
	}

	// Playoff weeks (Wildcard, Divisional, Conference, Super Bowl)
	playoffWeeks := []string{"1", "2", "3", "4"}
	for _, week := range playoffWeeks {
		periods = append(periods, PeriodInfo{
			Period:     week,
			PeriodType: "playoff", // Use lowercase to match existing main.go usage
		})
	}

	return periods
}
