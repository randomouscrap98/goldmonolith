package utils

import (
	"database/sql"
	"fmt"
)

// type DbConnectionOpener interface {
// 	OpenDb() (*sql.DB, error)
// }

type DbLike interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Given the create table / index commands, create all the tables and ALSO add a system
// value table which contains version information. You can use this table for other things too
func CreateTables_VersionedDb(allSql []string, db DbLike, version string) error {
	// db, err := conprov.OpenDb()
	// if err != nil {
	// 	return err
	// }
	// defer db.Close()

	allSql = append(allSql,
		`CREATE TABLE IF NOT EXISTS sysvalues (
      "key" TEXT PRIMARY KEY,
      value TEXT
    );`)

	for _, sql := range allSql {
		_, err := db.Exec(sql)
		if err != nil {
			return err
		}
	}

	_, err := db.Exec("INSERT OR IGNORE INTO sysvalues VALUES(?,?)", "version", version)
	if err != nil {
		return err
	}

	return nil
}

// Verify that a "versioned" database is the expected version
func VerifyVersionedDb(db DbLike, version string) error {
	var dbVersion string
	err := db.QueryRow("SELECT value FROM sysvalues WHERE \"key\" = ?", "version").Scan(&dbVersion)
	if err != nil {
		return err
	}
	if dbVersion != version {
		return fmt.Errorf("incompatible database version: expected %s, got %s", version, dbVersion)
	}
	return nil
}
