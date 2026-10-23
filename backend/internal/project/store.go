package project

import (
	"ppo-kanban/internal/db"
)

// Store defines database operations required by the project service.
// This interface allows mocking in unit tests while satisfying db.Querier.
type Store interface {
	db.Querier
}
