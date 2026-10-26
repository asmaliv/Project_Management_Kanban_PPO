package risk

import "ppo-kanban/internal/db"

// Store defines database operations required by the risk service.
type Store interface {
	db.Querier
}
