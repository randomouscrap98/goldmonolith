package kland

import (
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	DatabaseVersion = "1"
)

type Ban struct {
	range_  string //key?
	created time.Time
	note    *string
}

type Post struct {
	pid       int //key?
	created   time.Time
	content   string
	options   string
	ipaddress string
	username  *string
	tripraw   *string
	image     *string
	tid       int //Parent
}

type Thread struct {
	tid     int //key?
	created time.Time
	subject string
	deleted bool
	hash    *string

	// These are fields we query specially, but are still part of the thread query
	postCount  int
	lastPostOn time.Time
}

func CreateTables(config *Config) error {
	allSql := []string{
		`create table if not exists bans (
      range text unique,
      created text not null,
      note text
    );`,
		`create table if not exists threads (
      tid integer primary key,
      created text not null,
      subject text not null,
      deleted int not null,
      hash text
    );`,
		`create table if not exists posts (
      pid integer primary key,
      tid integer not null,
      created text not null,
      content text not null,
      options text not null,
      ipaddress text not null,
      username text,
      tripraw text,
      image text
    );`,
		`create index if not exists idx_posts_tid on posts(tid);`,
	}
	return utils.CreateTables_VersionedDb(allSql, config, DatabaseVersion)
}

// Pull ALL threads from the db.
func GetAllThreads(config *Config) ([]Thread, error) {
	db, err := config.OpenDb()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// The appending to this might suck idk
	result := make([]Thread, 0)

	// Go get the main data
	rows, err := db.Query(`
SELECT t.*,COUNT(p.pid),MAX(p.created) 
FROM threads t LEFT JOIN posts p ON t.tid = p.tid
WHERE t.deleted = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		thisThread := Thread{}
		err := rows.Scan()
		if err != nil {
			return nil, err
		}
		result = append(result, thisThread)
	}

	return result, nil
}
