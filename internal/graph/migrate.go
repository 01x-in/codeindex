package graph

import "fmt"

// MigrateStore runs all necessary migrations on the store.
// Currently only supports schema version 1.
func MigrateStore(store *SQLiteStore) error {
	if err := store.Migrate(); err != nil {
		return fmt.Errorf("running migration: %w", err)
	}
	return nil
}
