package kland

import (
	//"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/randomouscrap98/goldmonolith/utils"

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

func getTestDb(name string, t *testing.T) *sql.DB {
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
	db := getTestDb("basicopen", t)
	if db == nil {
		t.Fatalf("Database was nil on open")
	}
	db.Close()
}

func TestInsertBucketThread(t *testing.T) {
	db := getTestDb("insertthread", t)
	defer db.Close()
	tid, hash, err := InsertBucketThread(db, "hecking")
	if err != nil {
		t.Fatalf("Error on inserting bucket thread: %s", err)
	}
	thread, err := utils.FirstErr(GetThreadsById(db, []int64{tid}))
	if err != nil {
		t.Fatalf("Error retrieving bucket thread: %s", err)
	}
	if hash != thread.hash {
		t.Fatalf("Returned hash does not equal stored hash! %s vs %s", hash, thread.hash)
	}
	if len(thread.hash) != DbHashBaseCount {
		t.Fatalf("Hash not the right size! Expected %d, got %d", DbHashBaseCount, len(thread.hash))
	}
	if strings.Index(thread.subject, "hecking") < 0 {
		t.Fatalf("Subject malformed: %s", thread.subject)
	}
}

func TestUpdateThreadHash(t *testing.T) {
	db := getTestDb("updatebuckethash", t)
	defer db.Close()
	tid, hash, err := InsertBucketThread(db, "hecking")
	if err != nil {
		t.Fatalf("Error on inserting bucket thread: %s", err)
	}
	// Now we update
	hash2, err := UpdateThreadHash(db, tid)
	if err != nil {
		t.Fatalf("Error updating thread hash: %s", err)
	}
	if hash == hash2 {
		t.Fatalf("Hashes were supposed to be different: %s vs %s", hash2, hash)
	}
	// Go lookup the thread, it should have the new hash
	thread, err := utils.FirstErr(GetThreadsById(db, []int64{tid}))
	if err != nil {
		t.Fatalf("Error retrieving updated bucket thread: %s", err)
	}
	if thread.hash != hash2 {
		t.Fatalf("Updated bucket record hash didn't match: %s vs %s", thread.hash, hash2)
	}
}

func TestInsertManyBucketThreads(t *testing.T) {
	const REPEATS = 10000
	db := getTestDb("manybuckets", t)
	defer db.Close()
	hashes := make(map[string]int64)
	for i := range REPEATS {
		tid, hash, err := InsertBucketThread(db, fmt.Sprintf("hecking%d", i))
		if err != nil {
			t.Fatalf("Error on inserting bucket thread: %s", err)
		}
		oldtid, ok := hashes[hash]
		if ok {
			t.Fatalf("Repeat hash? %s on %d", hash, oldtid)
		}
		hashes[hash] = tid
	}
}
