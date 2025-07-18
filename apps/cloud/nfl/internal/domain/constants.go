package domain

// Season type constants for NFL seasons
const (
	PreSeason     = "1"
	RegularSeason = "2"
	PostSeason    = "3"
)

// SeasonTypeToNumber converts a season type string to its numeric representation
func SeasonTypeToNumber(seasonType string) string {
	switch seasonType {
	case PreSeason:
		return "1"
	case RegularSeason:
		return "2"
	case PostSeason:
		return "3"
	default:
		return "0"
	}
}
