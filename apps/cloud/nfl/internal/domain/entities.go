package domain

// Game represents an NFL game
type Game struct {
	ID         string
	Season     string
	Week       string
	SeasonType SeasonType
	HomeTeam   Team
	AwayTeam   Team
	Score      Score
}

// Team represents an NFL team
type Team struct {
	ID   string
	Name string
	Logo string
}

// Score represents a game's score
type Score struct {
	Home int
	Away int
}

// Rating represents a game's rating
type Rating struct {
	Score       int
	SpoilerFree string
	Explanation string
}

// SeasonType represents the type of season (regular, preseason, playoffs)
type SeasonType string

const (
	SeasonTypeRegular   SeasonType = "regular"
	SeasonTypePreseason SeasonType = "preseason"
	SeasonTypePlayoffs  SeasonType = "playoffs"
)

// Date represents a unique combination of season, week, and season type
type Date struct {
	Season     string
	Week       string
	SeasonType string
}

// Result represents a game result with its rating
type Result struct {
	ID         int
	EventID    string
	Season     string
	SeasonType string
	Week       string
	Rating     Rating
	Game       Game
}
