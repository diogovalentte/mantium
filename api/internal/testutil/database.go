//go:build db

package testutil

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/diogovalentte/mantium/api/src/db"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// Database creates an isolated database per test. The configured role needs CREATEDB.
func Database(t *testing.T) {
	t.Helper()
	admin, err := db.OpenConn()
	if err != nil {
		t.Fatal(err)
	}
	name := "mantium_test_" + uuid.New().String()
	if _, err = admin.Exec("CREATE DATABASE " + pq.QuoteIdentifier(name)); err != nil {
		t.Fatal(err)
	}
	original := os.Getenv("POSTGRES_DB")
	if err = db.CloseConn(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSTGRES_DB", name)
	t.Cleanup(func() {
		if err := db.CloseConn(); err != nil {
			t.Error(err)
		}
		if err := os.Setenv("POSTGRES_DB", original); err != nil {
			t.Error(err)
		}
		admin, err := db.OpenConn()
		if err != nil {
			t.Error(err)
			return
		}
		if _, err = admin.Exec("DROP DATABASE " + pq.QuoteIdentifier(name)); err != nil {
			t.Error(fmt.Errorf("drop test database: %w", err))
		}
		if err := db.CloseConn(); err != nil {
			t.Error(err)
		}
	})
	conn, err := db.OpenConn()
	if err != nil {
		t.Fatal(err)
	}
	log := zerolog.New(io.Discard)
	if err = db.CreateTables(conn, &log); err != nil {
		t.Fatal(err)
	}
}
