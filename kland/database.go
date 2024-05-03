package kland

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	DatabaseVersion    = "1"
	TimeFormat         = "2006-01-02 15:04:05" // Don't bother with the milliseconds
	HashBaseCount      = 5
	HashIncreaseFactor = 10000 // How many failures would require a base increase
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

// Generate a random thread hash that's never been used before. DOES NOT LOCK,
// you will need to do that!!
func GenerateThreadHash(db *sql.DB) (string, error) {
	retries := 0
	for {
		hash := utils.RandomAsciiName(HashBaseCount + retries/HashIncreaseFactor)
		// Go look for a thread with this hash. If one doesn't exist, we're good.
		threads, err := GetThreadsByField(db, "hash", hash)
		if err != nil {
			return "", err
		}
		if len(threads) == 0 {
			return hash, nil
		}
		retries += 1
	}
}

func QueryThreads(db *sql.DB, where func(string) string, limit func(string) string, params []any) ([]Thread, error) {
	// The appending to this might suck idk
	result := make([]Thread, 0)
	extrawhere := ""
	if where != nil {
		extrawhere = where("t")
	}
	extralimit := ""
	if limit != nil {
		extralimit = limit("t")
	}

	// Go get the main data
	rows, err := db.Query(fmt.Sprintf(`
SELECT t.tid, t.created, t.subject, t.deleted, t.hash, COUNT(p.pid), MAX(p.created) 
FROM threads t LEFT JOIN posts p ON t.tid = p.tid
%s
GROUP BY t.tid
%s
`, extrawhere, extralimit), params...)
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

func QueryPosts(db *sql.DB, where func(string) string, limit func(string) string, params []any) ([]Post, error) {
	result := make([]Post, 0)
	extrawhere := ""
	if where != nil {
		extrawhere = where("p")
	}
	extralimit := ""
	if limit != nil {
		extralimit = limit("p")
	}

	// Go get the main data
	rows, err := db.Query(fmt.Sprintf(`
SELECT p.pid, p.tid, p.created, p.content, p.options, p.ipaddress, p.username, p.tripraw, p.image
FROM posts p
%s
%s
`, extrawhere, extralimit), params...)
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

func orderTidDesc(t string) string {
	return fmt.Sprintf("ORDER BY %s.tid DESC", t)
}
func orderPid(t string) string {
	return fmt.Sprintf("ORDER BY %s.pid", t)
}

func GetAllThreads(db *sql.DB) ([]Thread, error) {
	return QueryThreads(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.deleted = 0", t)
		},
		orderTidDesc, nil)
}

func GetThreadsById(db *sql.DB, ids []int64) ([]Thread, error) {
	return QueryThreads(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.tid IN (%s)", t, utils.SliceToPlaceholder(ids))
		},
		orderTidDesc, utils.SliceToAny(ids))
}

func GetThreadsByField(db *sql.DB, field string, hash string) ([]Thread, error) {
	return QueryThreads(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.%s = ?", t, field)
		},
		orderTidDesc, []any{hash})
}

func GetPostsInThread(db *sql.DB, tid int64) ([]Post, error) {
	return QueryPosts(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.tid = ?", t)
		},
		orderPid, []any{tid})
}

func GetPaginatedPosts(db *sql.DB, tid int64, page int, perpage int) ([]Post, error) {
	return QueryPosts(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.tid = ?", t)
		},
		func(t string) string {
			return fmt.Sprintf("ORDER BY %s.pid DESC LIMIT ? OFFSET ?", t)
		}, []any{tid, perpage, perpage * page})
}
