package makai

import (
	"time"
)

type MySudokuOption struct {
	// Don't show your privates
	Default   interface{} `json:"default"`
	Value     interface{} `json:"value"`
	Title     string      `json:"title"`
	Possibles []string    `json:"possibles"`
}

func NewMySudokuOption(Default interface{}, Title string, Possibles []string) *MySudokuOption {
	return &MySudokuOption{
		Default:   Default,
		Value:     Default,
		Title:     Title,
		Possibles: Possibles,
	}
}

type PuzzleSetAggregate struct {
	PuzzleSet string `json:"puzzleset"`
	UID       int    `json:"uid"`
	Count     int    `json:"count"`
	Public    bool   `json:"public"`
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
	Number      int       `json:"number"`
	Pid         int       `json:"pid"`
	Completed   bool      `json:"completed"`
	Paused      bool      `json:"paused"`
	CompletedOn time.Time `json:"completedOn"`
	PausedOn    time.Time `json:"pausedOn"`
}

type QueryByPid struct { // this was derived from SDBPuzzle for some reason...
	PlayerSolution string `json:"playersolution"`
	Seconds        int    `json:"seconds"`
	Pid            int    `json:"pid"`
	Uid            int    `json:"uid"`
	Solution       string `json:"solution"`
	Puzzle         string `json:"puzzle"`
	PuzzleSet      string `json:"puzzleset"`
	Public         bool   `json:"public"`
}

type SudokuUser struct {
	// Normal fields
	Username string                    `json:"username"`
	Uid      int                       `json:"uid"`
	Admin    bool                      `json:"admin"`
	LoggedIn bool                      `json:"loggedIn"`
	Options  map[string]MySudokuOption `json:"-"`

	// Special field
	JsonOptions string `json:"-"`
	Exists      bool   `json:"exists"`
}
