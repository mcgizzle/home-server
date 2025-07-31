package domain

// Rating constants
const (
	// Rating types
	RatingTypeIDExcitement = "excitement"

	// Rating sources
	RatingSourceOpenAI = "openai"
)

// NFL-specific constants
const (
	// Period types for NFL
	PeriodTypeRegular = "Regular"
	PeriodTypePlayoff = "Playoff"

	// Home/Away designations
	HomeAwayHome = "home"
	HomeAwayAway = "away"

	// Competition status
	StatusScheduled  = "scheduled"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusCanceled   = "canceled"
)

// Rating categories for display
const (
	CategoryBoring    = "boring"
	CategoryOkay      = "okay"
	CategoryGood      = "good"
	CategoryGreat     = "great"
	CategoryAmazing   = "amazing"
	CategoryLegendary = "legendary"
)

// Rating category thresholds
var RatingCategoryThresholds = map[string]struct {
	Min int
	Max int
}{
	CategoryBoring:    {Min: 0, Max: 39},
	CategoryOkay:      {Min: 40, Max: 59},
	CategoryGood:      {Min: 60, Max: 74},
	CategoryGreat:     {Min: 75, Max: 84},
	CategoryAmazing:   {Min: 85, Max: 94},
	CategoryLegendary: {Min: 95, Max: 100},
}

// GetRatingCategory returns the category for a given rating score
func GetRatingCategory(score int) string {
	for category, threshold := range RatingCategoryThresholds {
		if score >= threshold.Min && score <= threshold.Max {
			return category
		}
	}
	return CategoryBoring // Default fallback
}
