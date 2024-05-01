package kland

import (
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	DatabaseVersion = "1"
	TimeFormat      = "2006-01-02 15:04:05" // Don't bother with the milliseconds
)

type Ban struct {
	range_  string //key?
	created string // FORMAT: 2023-01-02 HH:MM:SS.MS
	note    *string
}

type Post struct {
	pid       int    //key?
	created   string //time.Time
	content   string
	options   string
	ipaddress string
	username  *string
	tripraw   *string
	image     *string
	tid       int //Parent
}

type Thread struct {
	tid     int    //key?
	created string //time.Time
	subject string
	deleted bool
	hash    *string

	// These are fields we query specially, but are still part of the thread query
	postCount  int
	lastPostOn *string //time.Time
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

func parseTime(tstr string) time.Time {
	t, err := time.Parse(TimeFormat, tstr)
	if err != nil {
		// Horrible hack, I don't care. Kland sucks lmao
		return time.Date(1900, 01, 01, 0, 0, 0, 0, time.UTC)
	}
	return t
}

func parseTimePtr(tstr *string) time.Time {
	return parseTime(utils.Unpointer(tstr, ""))
}

// Pull threads from the db. Will pull ALL threads if id list is empty
func GetThreads(config *Config, ids []int64) ([]Thread, error) {
	db, err := config.OpenDb()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	extraWhere := ""
	anyIds := utils.SliceToAny(ids)

	if len(ids) > 0 {
		extraWhere = fmt.Sprintf("AND t.tid IN (%s)", utils.SliceToPlaceholder(ids))
	}

	// The appending to this might suck idk
	result := make([]Thread, 0)

	// Go get the main data
	rows, err := db.Query(fmt.Sprintf(`
SELECT t.tid, t.created, t.subject, t.deleted, t.hash, COUNT(p.pid), MAX(p.created) 
FROM threads t LEFT JOIN posts p ON t.tid = p.tid
WHERE t.deleted = 0 %s
GROUP BY t.tid
ORDER BY t.tid DESC
`, extraWhere), anyIds...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		t := Thread{}
		err := rows.Scan(&t.tid, &t.created, &t.subject, &t.deleted, &t.hash,
			&t.postCount, &t.lastPostOn)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	return result, nil
}

func GetPosts(tid int64, config *Config) ([]Post, error) {
	db, err := config.OpenDb()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	result := make([]Post, 0)

	// Go get the main data
	rows, err := db.Query(`
SELECT pid, tid, created, content, options, ipaddress, username, tripraw, image
FROM posts
WHERE tid = ?
ORDER BY pid
`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		p := Post{}
		err := rows.Scan(&p.pid, &p.tid, &p.created, &p.content, &p.options,
			&p.ipaddress, &p.username, &p.tripraw, &p.image)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}

	return result, nil
}
