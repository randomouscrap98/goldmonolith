package makai

import (
	"time"
)

// ONLY the basic data objects that make up sudoku.

type SudokuUserSession struct {
	UserId int64 `json:"uid"`
}

type SudokuSaveData struct {
	Puzzle string `json:"puzzle"`
}

type MySudokuOption struct {
	Default   interface{} `json:"default"`
	Value     interface{} `json:"value"`
	Title     string      `json:"title"`
	Possibles []string    `json:"possibles"`
}

type PuzzleSetAggregate struct {
	PuzzleSet string `json:"puzzleset" db:"puzzleset"`
	UID       int64  `json:"uid" db:"uid"`
	Count     int64  `json:"count" db:"count"`
	Public    bool   `json:"public" db:"public"`
}

type QueryObject struct {
	QueryOK      bool        `json:"queryok"`
	Result       interface{} `json:"result"`
	ResultIsLink bool        `json:"resultislink"`
	Errors       []string    `json:"errors"`
	Warnings     []string    `json:"warnings"`
	Requester    interface{} `json:"requester"`
}

type QueryByPuzzleset struct {
	Number      int        `json:"number" db:"number"`
	Pid         int64      `json:"pid" db:"pid"`
	Completed   bool       `json:"completed" db:"completed"`
	Paused      bool       `json:"paused" db:"paused"`
	CompletedOn *time.Time `json:"completedOn" db:"completedOn"`
	PausedOn    *time.Time `json:"pausedOn" db:"pausedOn"`
}

type QueryByPid struct { // this was derived from SDBPuzzle for some reason...
	PlayerSolution string `json:"playersolution" db:"playersolution"`
	Seconds        int    `json:"seconds" db:"seconds"` // Seconds for PAUSE, not complete
	Pid            int64  `json:"pid" db:"pid"`
	Uid            int64  `json:"uid" db:"uid"`
	Solution       string `json:"solution" db:"solution"`
	Puzzle         string `json:"puzzle" db:"puzzle"`
	PuzzleSet      string `json:"puzzleset" db:"puzzleset"`
	Public         bool   `json:"public" db:"public"`
}

type SudokuUser struct {
	// Normal fields
	Username string                     `json:"username"`
	Uid      int64                      `json:"uid"`
	Admin    bool                       `json:"admin"`
	LoggedIn bool                       `json:"loggedIn"`
	Options  map[string]*MySudokuOption `json:"-"`

	// Special field
	JsonOptions string `json:"-"`
	Exists      bool   `json:"exists"`
}

// -------- RAW DB OBJECTS -------------

// SDBUser represents the 'users' table
type SDBUser struct {
	UID          int64     `db:"uid"`
	Created      time.Time `db:"created"`
	Username     string    `db:"username"`
	Password     string    `db:"password"`
	SettingsJson string    `db:"settings"`
	Admin        bool      `db:"admin"`
}

// SDBSetting represents the 'settings' table
// type SDBSetting struct {
// 	SID   int64  `db:"sid"`
// 	UID   int64  `db:"uid"`
// 	Name  string `db:"name"`
// 	Value string `db:"value"`
// }

// SDBPuzzle represents the 'puzzles' table
type SDBPuzzle struct {
	PID       int64  `db:"pid"`
	UID       int64  `db:"uid"`
	Solution  string `db:"solution"`
	Puzzle    string `db:"puzzle"`
	PuzzleSet string `db:"puzzleset"`
	Public    bool   `db:"public"`
}

// SDBInProgress represents the 'inprogress' table
type SDBInProgress struct {
	IPID    int64     `db:"ipid"`
	UID     int64     `db:"uid"`
	PID     int64     `db:"pid"`
	Paused  time.Time `db:"paused"`
	Seconds int64     `db:"seconds"`
	Puzzle  string    `db:"puzzle"`
}

// SDBCompletions represents the 'completions' table
type SDBCompletions struct {
	CID       int64     `db:"cid"`
	UID       int64     `db:"uid"`
	PID       int64     `db:"pid"`
	Completed time.Time `db:"completed"`
	Seconds   int64     `db:"seconds"`
}
