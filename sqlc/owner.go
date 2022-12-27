package sqlc

import (
	"context"
	"log"

	"github.com/jackc/pgx/v4"
)

// RIPARTIRE QUI!<---
// - implicitl ycreate transaction?? 👈
func (q *Queries) WithOwner(tx pgx.Tx, owner string) *Queries {
	_, err := tx.Query(context.Background(), "set local tiny.owner = $1", owner)
	if err != nil {
		log.Fatal("failed to set owner on tx", owner)
	}
	return &Queries{
		db: tx,
	}
}
