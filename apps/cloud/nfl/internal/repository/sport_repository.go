package repository

// SportRepository handles sport-specific operations
type SportRepository interface {
	ListSports() ([]SportInfo, error)
	GetSport(sportID string) (SportInfo, error)
}

// SportInfo represents sport data from the database
type SportInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
