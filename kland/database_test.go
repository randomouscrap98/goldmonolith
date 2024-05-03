package kland

import (
	//"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	_ "github.com/mattn/go-sqlite3"
)

func GetTestConfig(t *testing.T) *Config {
	config_raw := GetDefaultConfig_Toml()
	var config Config
	err := toml.Unmarshal([]byte(config_raw), &config)
	if err != nil {
		t.Fatalf("Couldn't parse config toml: %s\n", err)
	}
	return &config
}

func GetTestDb(name string, t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", fmt.Sprintf(
		"file:kland_database_%s:?mode=memory&cache=shared&_busy_timeout=%d", name, BusyTimeout))
	if err != nil {
		t.Fatalf("Can't open database: %s", err)
	}
	err = CreateTables(db)
	if err != nil {
		db.Close()
		t.Fatalf("Can't create tables: %s", err)
	}
	return db
}

func TestOpenDb(t *testing.T) {
	db := GetTestDb("basicopen", t)
	if db == nil {
		t.Fatalf("Database was nil on open")
	}
	db.Close()
}

func TestInsertBucketThread(t *testing.T) {
	db := GetTestDb("insertthread", t)
	defer db.Close()
	thread, err := InsertBucketThread(db, "hecking")
	if err != nil {
		t.Fatalf("Error on inserting bucket thread: %s", err)
	}
	if len(*thread.hash) != HashBaseCount {
		t.Fatalf("Hash not the right size! Expected %d, got %d", HashBaseCount, len(*thread.hash))
	}
	if strings.Index(thread.subject, "hecking") < 0 {
		t.Fatalf("Subject malformed: %s", thread.subject)
	}
}

func TestUpdateThreadHash(t *testing.T) {
	db := GetTestDb("updatebuckethash", t)
	defer db.Close()
	thread, err := InsertBucketThread(db, "hecking")
	if err != nil {
		t.Fatalf("Error on inserting bucket thread: %s", err)
	}
	// Now we update
	thread2, err := UpdateThreadHash(db, thread.tid)
	if err != nil {
		t.Fatalf("Error updating thread hash: %s", err)
	}
	if thread2.hash == thread.hash {
		t.Fatalf("Hashes were supposed to be different: %s vs %s", *thread2.hash, *thread.hash)
	}
}

func TestInsertManyBucketThreads(t *testing.T) {
	const REPEATS = 1000
	db := GetTestDb("manybuckets", t)
	defer db.Close()
	hashes := make(map[string]struct{})
	var empty struct{}
	for i := range REPEATS {
		thread, err := InsertBucketThread(db, fmt.Sprintf("hecking%d", i))
		if err != nil {
			t.Fatalf("Error on inserting bucket thread: %s", err)
		}
		hash := *thread.hash
		_, ok := hashes[hash]
		if ok {
			t.Fatalf("Repeat hash? %s", hash)
		}
		hashes[hash] = empty
	}
}
