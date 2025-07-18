package domain

// Core business entities for the NFL ratings application

// Team represents a football team with its basic information
type Team struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Record string  `json:"record"`
	Logo   *string `json:"logo"`
}

// Game represents a complete NFL game with both teams and play details
type Game struct {
	Home    Team          `json:"home"`
	Away    Team          `json:"away"`
	Details []DetailsItem `json:"details"`
}

// Rating represents the AI-generated rant score and explanations
type Rating struct {
	Score       int    `json:"score"`
	Explanation string `json:"explanation"`
	SpoilerFree string `json:"spoiler_free_explanation"`
}

// Result represents a complete game result with metadata and rating
type Result struct {
	Id         int    `json:"id"`
	EventId    string `json:"event_id"`
	Season     string `json:"season"`
	SeasonType string `json:"season_type"`
	Week       string `json:"week"`
	Rating     Rating `json:"rating"`
	Game       Game   `json:"game"`
}

// Date represents a specific NFL week/season combination
type Date struct {
	Season     string
	Week       string
	SeasonType string
}

// DateTemplate represents a date formatted for display in the UI
type DateTemplate struct {
	Season             string
	Week               string
	SeasonType         string
	SeasonTypeShowable string
}

// TemplateData represents the complete data structure passed to HTML templates
type TemplateData struct {
	Results []Result
	Dates   []DateTemplate
	Current DateTemplate
}

// DetailsItem represents a single play or event in the game
type DetailsItem struct {
	ShortText    string  `json:"shortText"`
	ScoringPlay  bool    `json:"scoringPlay"`
	ScoringValue float64 `json:"scoringValue"`
	Clock        struct {
		DisplayValue string `json:"displayValue"`
	}
}

// Methods for Date entity
func (d Date) Template() DateTemplate {
	var seasonType string
	switch d.SeasonType {
	case "1":
		seasonType = "Preseason"
	case "2":
		seasonType = "Regular Season"
	case "3":
		seasonType = "Postseason"
	}

	return DateTemplate{
		Season:             d.Season,
		Week:               d.Week,
		SeasonTypeShowable: seasonType,
		SeasonType:         d.SeasonType,
	}
}
