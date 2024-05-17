package makai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

func (mctx *MakaiContext) GetLoggedInSudokuUid(r *http.Request) (int64, error) {
	cookie, err := r.Cookie(SudokuCookie)
	if err != nil {
		if err == http.ErrNoCookie {
			return -1, nil
		} else {
			return 0, err
		}
	}
	session, err := mctx.GetSudokuSession(cookie.Value)
	if err != nil {
		return 0, err
	}
	return session.UserId, nil
}

// Using the request + cookie + whatever, get the fully filled out current sudoku user.
// Immediately usable in rendering and whatever else you want.
func (mctx *MakaiContext) GetLoggedInSudokuUser(r *http.Request) (*SudokuUser, error) {
	uid, err := mctx.GetLoggedInSudokuUid(r)
	if err != nil {
		return nil, err
	}
	rawuser, err := mctx.GetSudokuUserById(uid)
	if err != nil {
		return nil, err
	}
	user := rawuser.ToUser(true)
	return &user, nil
}

func (mctx *MakaiContext) SudokuQueryWeb(r *http.Request) (string, error) {
	uid, err := mctx.GetLoggedInSudokuUid(r)
	if err != nil {
		log.Printf("Error retrieving logged in sudoku user: %s", err)
	}
	err = r.ParseMultipartForm(int64(mctx.config.MaxFormMemory))
	if err != nil {
		return "", fmt.Errorf("Couldn't parse form: %s", err)
		// result(queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err)))
		// return
	}
	puzzleset := r.FormValue("puzzleset")
	pid := r.FormValue("pid")
	if puzzleset != "" {
		puzzles, err := mctx.GetPuzzlesetData(puzzleset, uid)
		if err != nil {
			return "", fmt.Errorf("Error retrieving puzzles: %s", err)
			// log.Printf("Error retrieving puzzles: %s", err)
			// result(queryFromErrors(fmt.Sprintf("Error retrieving puzzles: %s", err)))
			// return
		}
		puzzleresult, err := json.Marshal(puzzles)
		if err != nil {
			return "", fmt.Errorf("Error jsoning puzzles: %s", err)
			// log.Printf("Error jsoning puzzles: %s", err)
			// result(queryFromErrors(fmt.Sprintf("Error jsoning puzzles: %s", err)))
			// return
		}
		return string(puzzleresult), nil
		//result(queryFromResult(puzzleresult))
	} else if pid != "" {
		pidint, err := strconv.ParseInt(pid, 10, 64)
		if err != nil {
			return "", err
			// result(queryFromErrors("Invalid pid"))
			// return
		}
		puzzle, err := mctx.GetPuzzle(pidint, uid)
		if err != nil {
			return "", fmt.Errorf("Error retrieving puzzle: %s", err)
			//log.Printf("Error retrieving puzzle: %s", err)
			//result(queryFromErrors(fmt.Sprintf("Error retrieving puzzle: %s", err)))
			//return
		}
		puzzleresult, err := json.Marshal(puzzle)
		if err != nil {
			return "", fmt.Errorf("Error jsoning puzzle: %s", err)
			// log.Printf("Error jsoning puzzle: %s", err)
			// result(queryFromErrors(fmt.Sprintf("Error jsoning puzzle: %s", err)))
			// return
		}
		return string(puzzleresult), nil
		//result(queryFromResult(puzzleresult))
	} else {
		return "", fmt.Errorf("You must provider EITHER pid OR puzzleset!")
		//result(queryFromErrors("You must provide EITHER pid OR puzzleset!"))
	}
}
