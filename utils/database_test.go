package utils

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const BusyTimeout = 5000

func getTestDb(name string, t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", fmt.Sprintf(
		"file:utils_database_%s:?mode=memory&cache=shared&_busy_timeout=%d", name, BusyTimeout))
	if err != nil {
		t.Fatalf("Can't open database: %s", err)
	}
	return db
}

func TestVersionedDatabaseSimple(t *testing.T) {
	db := getTestDb("versioned_same", t)
	defer db.Close()
	err := CreateTables_VersionedDb([]string{}, db, "0.1")
	if err != nil {
		t.Fatalf("Can't create versioned database: %s", err)
	}
	err = VerifyVersionedDb(db, "0.1")
	if err != nil {
		t.Fatalf("Expected versions to be the same, got: %s", err)
	}
	err = VerifyVersionedDb(db, "0.2")
	if err == nil {
		t.Fatalf("Expected versions to not be the same")
	}
}

func TestVersionedDatabaseExtraTables(t *testing.T) {
	db := getTestDb("versioned_same", t)
	defer db.Close()
	err := CreateTables_VersionedDb([]string{
		"CREATE TABLE junk(pid INTEGER PRIMARY KEY, val TEXT)",
		"INSERT INTO JUNK(val) VALUES('heck')",
		"CREATE TABLE junk2(zid INTEGER PRIMARY KEY, pid INT NOT NULL)",
		`CREATE INDEX idx_junk ON junk(val)`,
	}, db, "0.1")
	if err != nil {
		t.Fatalf("Can't create versioned database: %s", err)
	}
	err = VerifyVersionedDb(db, "0.1")
	if err != nil {
		t.Fatalf("Expected versions to be the same, got: %s", err)
	}
	err = VerifyVersionedDb(db, "0.1.1")
	if err == nil {
		t.Fatalf("Expected versions to not be the same")
	}
}
