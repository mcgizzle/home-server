package domain

import (
	"time"
)

// Sport represents the type of sport - now data-driven instead of hardcoded
type Sport string

// Competition represents a universal competition/game across any sport
type Competition struct {
	ID         string              `json:"id"`
	EventID    string              `json:"event_id"`
	Sport      Sport               `json:"sport"`
	Season     string              `json:"season"`
	Period     string              `json:"period"`      // Week/Game/Round
	PeriodType string              `json:"period_type"` // Regular/Playoff/etc
	StartTime  *time.Time          `json:"start_time,omitempty"`
	Status     string              `json:"status"`
	Teams      []CompetitionTeam   `json:"teams"`
	Rating     *Rating             `json:"rating,omitempty"`
	Details    *CompetitionDetails `json:"details,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
}

// ToTemplateResult converts a V2 Competition to TemplateResult for UI display
func (c Competition) ToTemplateResult() TemplateResult {
	templateRating := TemplateRating{
		Score:       0,
		Explanation: "",
		SpoilerFree: "",
		Category:    "unrated",
	}

	if c.Rating != nil {
		templateRating = TemplateRating{
			Score:       c.Rating.Score,
			Explanation: c.Rating.Explanation,
			SpoilerFree: c.Rating.SpoilerFree,
			Category:    getRatingCategory(c.Rating.Score),
		}
	}

	return TemplateResult{
		ID:          c.ID,
		EventID:     c.EventID,
		Season:      c.Season,
		Period:      c.Period,
		PeriodType:  c.PeriodType,
		Rating:      templateRating,
		Competition: c,
	}
}

// Team represents a team across any sport
type Team struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Sport   Sport  `json:"sport"`
	LogoURL string `json:"logo_url,omitempty"`
}

// CompetitionTeam represents a team's participation in a specific competition
type CompetitionTeam struct {
	Team     Team                   `json:"team"`
	HomeAway string                 `json:"home_away"` // "home" or "away"
	Score    float64                `json:"score"`
	Stats    map[string]interface{} `json:"stats,omitempty"` // Sport-specific stats
}

// RatingType represents the type of rating (excitement only initially)
type RatingType string

const (
	RatingTypeExcitement RatingType = "excitement" // Only excitement rating initially
)

// Rating represents an AI-generated rating with metadata
type Rating struct {
	Type        RatingType `json:"type"`
	Score       int        `json:"score"`
	Confidence  float64    `json:"confidence,omitempty"`
	Explanation string     `json:"explanation"`
	SpoilerFree string     `json:"spoiler_free"`
	Source      string     `json:"source"`
	GeneratedAt time.Time  `json:"generated_at"`
}

// CompetitionDetails represents rich competition data like play-by-play
type CompetitionDetails struct {
	PlayByPlay []interface{}          `json:"play_by_play,omitempty"` // JSON array
	Metadata   map[string]interface{} `json:"metadata,omitempty"`     // Sport-specific data
}

// SentimentRating represents an AI-generated sentiment analysis from fan reactions
type SentimentRating struct {
	Source       string    `json:"source"`        // "reddit", "twitter", etc.
	ThreadURL    string    `json:"thread_url"`    // URL to the source thread
	CommentCount int       `json:"comment_count"` // Number of comments analyzed
	Score        int       `json:"score"`         // 0-100 excitement/engagement rating
	Sentiment    string    `json:"sentiment"`     // excited, neutral, disappointed
	Highlights   []string  `json:"highlights"`    // Key themes from the comments
	GeneratedAt  time.Time `json:"generated_at"`  // When the analysis was performed
}

// Result represents a complete game result with metadata (compatibility type)
// This mirrors the v1 Result struct for easier migration
type Result struct {
	Competition Competition `json:"competition"`
}

// Date represents a period identifier for querying competitions
type Date struct {
	Season     string `json:"season"`
	Period     string `json:"period"`      // Week/Game/Round
	PeriodType string `json:"period_type"` // Regular/Playoff/etc
}

// Template converts a V2 Date to DateTemplate for UI display
func (d Date) Template() DateTemplate {
	return DateTemplate{
		Season:             d.Season,
		Period:             d.Period,
		PeriodDisplay:      getPeriodDisplay(d.Period, d.PeriodType),
		PeriodTypeShowable: getPeriodTypeDisplay(d.PeriodType),
		PeriodType:         d.PeriodType,
	}
}

// DateTemplate represents a date formatted for UI display
type DateTemplate struct {
	Season             string `json:"season"`
	Period             string `json:"period"`
	PeriodDisplay      string `json:"period_display"`
	PeriodTypeShowable string `json:"period_type_showable"`
	PeriodType         string `json:"period_type"`
}

// TemplateRating represents a rating with computed category for template display
type TemplateRating struct {
	Score       int    `json:"score"`
	Explanation string `json:"explanation"`
	SpoilerFree string `json:"spoiler_free"`
	Category    string `json:"category"`
}

// TemplateSentiment represents sentiment data formatted for template display
type TemplateSentiment struct {
	Score        int      `json:"score"`
	Sentiment    string   `json:"sentiment"` // excited, neutral, disappointed
	Highlights   []string `json:"highlights"`
	CommentCount int      `json:"comment_count"`
	ThreadURL    string   `json:"thread_url"`
	HasData      bool     `json:"has_data"` // True if sentiment data exists
}

// TemplateResult represents a result formatted for template display
type TemplateResult struct {
	ID          string            `json:"id"`
	EventID     string            `json:"event_id"`
	Season      string            `json:"season"`
	Period      string            `json:"period"`
	PeriodType  string            `json:"period_type"`
	Rating      TemplateRating    `json:"rating"`
	Sentiment   TemplateSentiment `json:"sentiment"`
	Competition Competition       `json:"competition"`
}

// TemplateData represents data formatted for web template rendering
type TemplateData struct {
	Results []TemplateResult `json:"results"`
	Dates   []DateTemplate   `json:"dates"`   // All available dates
	Seasons []string         `json:"seasons"` // All available seasons
	Current DateTemplate     `json:"current"` // Currently selected date
}

// Helper functions

func getRatingCategory(score int) string {
	switch {
	case score >= 90:
		return "must-watch"
	case score >= 70:
		return "recommended"
	case score >= 50:
		return "good"
	case score >= 30:
		return "okay"
	default:
		return "skippable"
	}
}

func getPeriodTypeDisplay(periodType string) string {
	switch periodType {
	case "regular":
		return "Regular Season"
	case "playoff":
		return "Playoffs"
	case "preseason":
		return "Preseason"
	default:
		return periodType
	}
}

// getPeriodDisplay converts a period number to a display name, especially for playoff rounds
// This mirrors the V1 WeekToDisplay function to ensure consistency
func getPeriodDisplay(period string, periodType string) string {
	if periodType == "playoff" { // Playoffs
		switch period {
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
			return "Week " + period
		}
	}
	return "Week " + period
}
