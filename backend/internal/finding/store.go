package finding

import "ppo-kanban/internal/db"

// Store defines database operations required by the finding service.
type Store interface {
	db.Querier
}
