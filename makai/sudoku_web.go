package makai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// All sudoku "wrapper" functions specifically made for the web portion of sudoku.
// These are the kinds of things that might go into a "controller" from dotnet,
// while the other sudoku file is for the inner workings of sudoku, which can be
// used and tested without web context

const (
	SudokuCookie = "makai_sudoku_session"
)

type SudokuLoginQuery struct {
	Username  string `schema:"username"`
	Password  string `schema:"password"`
	Password2 string `schema:"password2"`
	Logout    bool   `schema:"logout"`
}

func (mctx *MakaiContext) RenderSudoku(subtemplate string, w http.ResponseWriter, r *http.Request) {
	data := mctx.GetDefaultData(r)
	data["oroot"] = mctx.config.RootPath + "/sudoku"
	data["template_"+subtemplate] = true
	data["debug"] = r.URL.Query().Has("debug")
	data["puzzleSets"] = "" // Some serialized thing...
	_, err := os.Stat(mctx.config.SudokuDbPath)
	data["dbexists"] = err == nil
	user, err := mctx.GetLoggedInSudokuUser(r)
	if err != nil {
		log.Printf("Error getting user session: %s", err)
	} else if user != nil {
		data["user"] = user
	}
	data["puzzleSets"], err = mctx.GetJsonPuzzleSetsForUser(user)
	if err != nil {
		// this is a fullstop error
		http.Error(w, fmt.Sprintf("Couldn't load puzzles: %s", err), http.StatusInternalServerError)
		return
	}

	mctx.RunTemplate("sudoku_index.tmpl", w, data)
}

func (mctx *MakaiContext) GetJsonPuzzleSetsForUser(user *SudokuUser) (string, error) {
	var uid int64
	if user != nil {
		uid = user.Uid
	}
	puzzles, err := mctx.GetPuzzleSets(uid)
	if err != nil {
		return "", err
	}
	puzzlesjson, err := json.Marshal(puzzles)
	if err != nil {
		return "", err
	}
	return string(puzzlesjson), nil
}

// Using the request + cookie + whatever, get the fully filled out current sudoku user.
// Immediately usable in rendering and whatever else you want.
func (mctx *MakaiContext) GetLoggedInSudokuUser(r *http.Request) (*SudokuUser, error) {
	cookie, err := r.Cookie(SudokuCookie)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		} else {
			return nil, err
		}
	}
	rawuser, err := mctx.GetSudokuSession(cookie.Value)
	if err != nil {
		return nil, err
	}
	user, err := rawuser.ToUser(true)
	return &user, err
}