package repository

import (
	"github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"
)

// TeamRepository handles team-specific operations
type TeamRepository interface {
	SaveTeam(team domain.Team) error
}
