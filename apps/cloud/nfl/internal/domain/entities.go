package domain

import (
	"encoding/json"
)

// Team represents a football team with its basic information
type Team struct {
	Name   string  `json:"name"`
	Logo   *string `json:"logo,omitempty"`
	Score  float64 `json:"score"`
	Record string  `json:"record"`
}

// Game represents a complete NFL game with both teams and play details
type Game struct {
	Home    Team          `json:"home"`
	Away    Team          `json:"away"`
	Details []DetailsItem `json:"details"`
}

// Rating represents an AI-generated rant score and explanations
type Rating struct {
	Score       int    `json:"score"`
	Explanation string `json:"explanation"`
	SpoilerFree string `json:"spoiler_free_explanation"`
}

// RatingCategory represents a categorization of game ratings
type RatingCategory struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Min         int    `json:"min_score"`
	Max         int    `json:"max_score"`
}

// Result represents a complete game result with metadata
type Result struct {
	Id         int    `json:"id"`
	EventId    string `json:"event_id"`
	Season     string `json:"season"`
	Week       string `json:"week"`
	SeasonType string `json:"season_type"`
	Rating     Rating `json:"rating"`
	Game       Game   `json:"game"`
}

// Date represents an NFL week/season combination
type Date struct {
	Season     string `json:"season"`
	Week       string `json:"week"`
	SeasonType string `json:"season_type"`
}

// DateTemplate represents a date formatted for UI display
type DateTemplate struct {
	Season             string `json:"season"`
	Week               string `json:"week"`
	WeekDisplay        string `json:"week_display"`
	SeasonTypeShowable string `json:"season_type_showable"`
	SeasonType         string `json:"season_type"`
}

// TemplateRating represents a rating with computed category for template display
type TemplateRating struct {
	Score       int            `json:"score"`
	Explanation string         `json:"explanation"`
	SpoilerFree string         `json:"spoiler_free_explanation"`
	Category    RatingCategory `json:"category"`
}

// TemplateResult represents a result with template-specific enhancements
type TemplateResult struct {
	Id         int            `json:"id"`
	EventId    string         `json:"event_id"`
	Season     string         `json:"season"`
	Week       string         `json:"week"`
	SeasonType string         `json:"season_type"`
	Rating     TemplateRating `json:"rating"`
	Game       Game           `json:"game"`
}

// TemplateData represents the complete data structure passed to HTML templates
type TemplateData struct {
	Results []TemplateResult `json:"results"`
	Dates   []DateTemplate   `json:"dates"`
	Seasons []string         `json:"seasons"`
	Current DateTemplate     `json:"current"`
}

// ToTemplateResult converts a domain Result to a TemplateResult with computed category
func (r Result) ToTemplateResult() TemplateResult {
	return TemplateResult{
		Id:         r.Id,
		EventId:    r.EventId,
		Season:     r.Season,
		Week:       r.Week,
		SeasonType: r.SeasonType,
		Rating: TemplateRating{
			Score:       r.Rating.Score,
			Explanation: r.Rating.Explanation,
			SpoilerFree: r.Rating.SpoilerFree,
			Category:    GetRatingCategory(r.Rating.Score),
		},
		Game: r.Game,
	}
}

// DetailsItem represents individual plays or events in games
type DetailsItem struct {
	Text string `json:"text"`
}

// Template converts a Date to a DateTemplate for UI display
func (d Date) Template() DateTemplate {
	seasonTypeShowable := SeasonTypeToDisplay(d.SeasonType)
	weekDisplay := WeekToDisplay(d.Week, d.SeasonType)
	return DateTemplate{
		Season:             d.Season,
		Week:               d.Week,
		WeekDisplay:        weekDisplay,
		SeasonTypeShowable: seasonTypeShowable,
		SeasonType:         d.SeasonType,
	}
}

// SeasonTypeToDisplay converts season type numbers to display names
func SeasonTypeToDisplay(seasonType string) string {
	switch seasonType {
	case "1":
		return "Pre-Season"
	case "2":
		return "Regular Season"
	case "3":
		return "Post-Season"
	default:
		return "Unknown"
	}
}

// WeekToDisplay converts a week number to a display name, especially for post-season rounds
func WeekToDisplay(week string, seasonType string) string {
	if seasonType == "3" { // Post-Season
		switch week {
		case "1":
			return "Wild Card"
		case "2":
			return "Divisional"
		case "3":
			return "Conference Championship"
		case "4":
			return "Pro Bowl"
		case "5":
			return "Super Bowl"
		default:
			return "Week " + week
		}
	}
	return "Week " + week
}

// SeasonTypeToDisplayWithRounds converts season type numbers to display names, including post-season rounds
func SeasonTypeToDisplayWithRounds(seasonType string, week string) string {
	switch seasonType {
	case "1":
		return "Pre-Season"
	case "2":
		return "Regular Season"
	case "3":
		return WeekToDisplay(week, seasonType)
	default:
		return "Unknown"
	}
}

// UnmarshalJSON custom unmarshaling for Rating to handle the JSON structure
func (r *Rating) UnmarshalJSON(data []byte) error {
	var temp struct {
		Score       int    `json:"score"`
		Explanation string `json:"explanation"`
		SpoilerFree string `json:"spoiler_free_explanation"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	r.Score = temp.Score
	r.Explanation = temp.Explanation
	r.SpoilerFree = temp.SpoilerFree
	return nil
}

// MarshalJSON custom marshaling for Rating to ensure proper JSON structure
func (r Rating) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Score       int    `json:"score"`
		Explanation string `json:"explanation"`
		SpoilerFree string `json:"spoiler_free_explanation"`
	}{
		Score:       r.Score,
		Explanation: r.Explanation,
		SpoilerFree: r.SpoilerFree,
	})
}

// GetRatingCategory returns the appropriate rating category for a given score
func GetRatingCategory(score int) RatingCategory {
	categories := []RatingCategory{
		{Key: "legendary", Name: "Legendary", Description: "Instant classic", Min: 90, Max: 100},
		{Key: "must_watch", Name: "Must Watch", Description: "Don't miss this one", Min: 75, Max: 89},
		{Key: "entertaining", Name: "Entertaining", Description: "Worth your time", Min: 60, Max: 74},
		{Key: "decent", Name: "Decent", Description: "Solid game", Min: 45, Max: 59},
		{Key: "mediocre", Name: "Mediocre", Description: "Only if you're bored", Min: 30, Max: 44},
		{Key: "skippable", Name: "Skippable", Description: "Save your time", Min: 15, Max: 29},
		{Key: "unwatchable", Name: "Unwatchable", Description: "Complete snoozefest", Min: 0, Max: 14},
	}

	for _, category := range categories {
		if score >= category.Min && score <= category.Max {
			return category
		}
	}

	// Fallback for edge cases (should not happen with valid scores 0-100)
	return categories[len(categories)-1] // Return unwatchable as fallback
}
