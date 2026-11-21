package db

import (
	"database/sql"
	"testing"
)

// TestMigrationsLoad checks that goose accepts the embedded migrations' names
// and versions, without a database.
func TestMigrationsLoad(t *testing.T) {
	// sql.Open doesn't connect, and neither does loading the migrations
	sqlDB, err := sql.Open("pgx", "postgres://localhost/unused")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	provider, err := newMigrationProvider(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	sources := provider.ListSources()
	if len(sources) == 0 || sources[0].Version != 1 {
		t.Errorf("migrations = %v", sources)
	}
}
