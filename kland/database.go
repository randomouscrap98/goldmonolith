package kland

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	DatabaseVersion      = "1"
	TimeFormat           = "2006-01-02 15:04:05" // Don't bother with the milliseconds
	DbHashBaseCount      = 5
	DbHashIncreaseFactor = 100 // How many failures would require a base increase
	OrphanedPrepend      = "Internal_OrphanedImages"
	OrphanedPostContent  = "orphanedPost"
)

// NOTE: Ban isn't used
type Ban struct {
	range_  string //key?
	created string // FORMAT: 2023-01-02 HH:MM:SS.MS
	note    string // was nullable
}

type Post struct {
	Pid       int64  //key?
	Created   string // time.Time in TimeFormat format
	Content   string
	Options   string
	Ipaddress string
	Username  string // nullable in db
	Tripraw   string // nullable in db
	Image     string // nullable in db
	Tid       int64  // Parent thread
}

type Thread struct {
	Tid     int64  //key?
	Created string // time.Time in TimeFormat format
	Subject string
	Deleted bool
	Hash    string // nullable in db

	// These are fields we query specially, but are still part of the thread query
	PostCount  int
	LastPostOn string //time.Time
}

func CreateTables(db utils.DbLike) error {
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
		`create table if not exists rehashes (
      rid integer primary key,
      oldhash text not null,
      newhash next not null
    );`,
		`create index if not exists idx_threads_subject on threads(subject);`,
		`create index if not exists idx_threads_hash on threads(hash);`,
		`create index if not exists idx_posts_tid on posts(tid);`,
		`create index if not exists idx_rehashes_oldhash_newhash on rehashes(oldhash, newhash);`,
	}
	return utils.CreateTables_VersionedDb(allSql, db, DatabaseVersion)
}

// Parse a KLAND time, because they have a weird format...
func parseTime(tstr string) time.Time {
	t, err := time.Parse(TimeFormat, tstr)
	if err != nil {
		// Horrible hack, I don't care. Kland sucks lmao
		return time.Date(1900, 01, 01, 0, 0, 0, 0, time.UTC)
	}
	return t
}

func BucketSubject(bucket string) string {
	if bucket == "" {
		return OrphanedPrepend
	} else {
		return fmt.Sprintf("%s_%s", OrphanedPrepend, bucket)
	}
}

// Generate a random thread hash that's never been used before. DOES NOT LOCK,
// you will need to do that!!
func GenerateThreadHash(db utils.DbLike) (string, error) {
	retries := 0
	var count int
	for {
		hash := utils.RandomAsciiName(DbHashBaseCount + retries/DbHashIncreaseFactor)
		// Go look for a thread with this hash. If one doesn't exist, we're good.
		err := db.QueryRow("SELECT COUNT(*) FROM threads WHERE hash = ?", hash).Scan(&count)
		if err != nil {
			return "", err
		}
		if count == 0 {
			return hash, nil
		}
		retries += 1
	}
}

// Update hash on thread to another random value. Return the new hash
func UpdateThreadHash(db utils.DbLike, tid int64) (string, error) {
	hash, err := GenerateThreadHash(db)
	if err != nil {
		return "", err
	}
	_, err = db.Exec("UPDATE threads SET hash=? WHERE tid=?", hash, tid)
	if err != nil {
		return "", err
	}
	return hash, nil //utils.FirstErr(GetThreadsByField(db, "tid", tid))
}

// Add a bucket thread, generating a random hash. Returns the id of the
// inserted thread and the hash
func InsertBucketThread(db utils.DbLike, subject string) (int64, string, error) {
	hash, err := GenerateThreadHash(db)
	if err != nil {
		return 0, "", err
	}
	result, err := db.Exec("INSERT INTO threads(subject, created, deleted, hash) VALUES (?,?,?,?)",
		subject, time.Now().Format(TimeFormat), true, hash)
	if err != nil {
		return 0, "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, "", err
	}
	return id, hash, nil
}

// Add a post from the given ip with the given file to the given thread. Returns the
// id of the post as inserted
func InsertImagePost(db utils.DbLike, ip string, filename string, tid int64) (int64, error) {
	result, err := db.Exec("INSERT INTO posts(content, created, ipaddress, image, tid, options) VALUES (?,?,?,?,?,?)",
		OrphanedPostContent, time.Now().Format(TimeFormat), ip, filename, tid, "")
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// Generic thread query. You determine the where and limiting clauses, and this function
// runs and parses the query. You can probably get a library to do this...
func QueryThreads(db utils.DbLike, where func(string) string, limit func(string) string, params []any) ([]Thread, error) {
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
SELECT 
  t.tid, 
  t.created, 
  t.subject, 
  t.deleted, 
  COALESCE(t.hash,''), 
  COUNT(p.pid), 
  COALESCE(MAX(p.created),'')
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
		err := rows.Scan(&t.Tid, &t.Created, &t.Subject, &t.Deleted, &t.Hash,
			&t.PostCount, &t.LastPostOn)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	return result, nil
}

// Generic post query. You determine the where and limiting clauses, and this function
// runs and parses the query. You can probably get a library to do this...
func QueryPosts(db utils.DbLike, where func(string) string, limit func(string) string, params []any) ([]Post, error) {
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
SELECT 
  p.pid, 
  p.tid, 
  p.created, 
  p.content, 
  p.options, 
  p.ipaddress, 
  COALESCE(p.username,''), 
  COALESCE(p.tripraw,''),
  COALESCE(p.image,'')
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
		err := rows.Scan(&p.Pid, &p.Tid, &p.Created, &p.Content, &p.Options,
			&p.Ipaddress, &p.Username, &p.Tripraw, &p.Image)
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

func GetAllThreads(db utils.DbLike) ([]Thread, error) {
	return QueryThreads(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.deleted = 0", t)
		},
		orderTidDesc, nil)
}

func GetThreadsById(db utils.DbLike, ids []int64) ([]Thread, error) {
	return QueryThreads(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.tid IN (%s)", t, utils.SliceToPlaceholder(ids))
		},
		orderTidDesc, utils.SliceToAny(ids))
}

func GetThreadsByField(db utils.DbLike, field string, value any) ([]Thread, error) {
	return QueryThreads(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.%s = ?", t, field)
		},
		orderTidDesc, []any{value})
}

func GetPostsInThread(db utils.DbLike, tid int64) ([]Post, error) {
	return QueryPosts(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.tid = ?", t)
		},
		orderPid, []any{tid})
}

func GetPaginatedPosts(db utils.DbLike, tid int64, page int, perpage int) ([]Post, error) {
	return QueryPosts(db,
		func(t string) string {
			return fmt.Sprintf("WHERE %s.tid = ?", t)
		},
		func(t string) string {
			return fmt.Sprintf("ORDER BY %s.pid DESC LIMIT ? OFFSET ?", t)
		}, []any{tid, perpage, perpage * page})
}

func AddRehash(db *sql.DB, p *Post, newimage string, newtag string) error {
	// Start a transaction for the two updates we're going to do
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Now, update the post and add to the rehash. The transaction will
	// get rid of anything that gets left dangling
	_, err = tx.Exec("INSERT INTO rehashes(oldhash, newhash) VALUES(?,?)", p.Image, newimage)
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE posts SET image=?, username=? WHERE pid=?", newimage, newtag, p.Pid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Simply return the newhash based on the old hash, if it exists...
func LookupRehash(db utils.DbLike, hash string) (string, error) {
	var newhash string
	err := db.QueryRow("select newhash from rehashes where oldhash=?", hash).Scan(&newhash)
	return newhash, err
}
