package postgres

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TxFn is a function that runs inside a database transaction.
type TxFn func(tx *gorm.DB) error

// RunInTransaction executes fn inside a transaction. It commits when fn returns
// nil and rolls back when fn returns an error. The passed *gorm.DB is the
// transaction-scoped DB and should be used for all queries inside fn.
func RunInTransaction(ctx context.Context, db *gorm.DB, fn TxFn) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fn(tx); err != nil {
			return fmt.Errorf("transaction failed: %w", err)
		}
		return nil
	})
}
