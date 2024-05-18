package makai

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/kataras/jwt"
	"golang.org/x/crypto/bcrypt"

	"github.com/randomouscrap98/goldmonolith/utils"
)

func queryFromResult(result any) QueryObject {
	return QueryObject{
		QueryOK: true,
		Result:  result,
	}
}

func queryFromErrors(errors ...string) QueryObject {
	return QueryObject{
		QueryOK: false,
		Errors:  errors,
		Result:  false,
	}
}

func queryFromError(err error) QueryObject {
	_, ok := err.(*utils.ExpectedError)
	if ok {
		return queryFromErrors(err.Error())
	} else {
		log.Printf("INTERNAL SERVER ERROR: %s", err)
		return queryFromErrors("Internal server error")
	}
}

func newMySudokuOption(Default interface{}, Title string, Possibles []string) *MySudokuOption {
	if Possibles == nil {
		Possibles = make([]string, 0)
	}
	return &MySudokuOption{
		Default:   Default,
		Value:     Default,
		Title:     Title,
		Possibles: Possibles,
	}
}

// Get all options available and their defaults
func getDefaultSudokuOptions() map[string]*MySudokuOption {
	result := make(map[string]*MySudokuOption)
	result["lowperformance"] = newMySudokuOption(false, "Low Performance Mode", nil)
	result["completed"] = newMySudokuOption(true, "Disable buttons for completed numbers", nil)
	result["noteremove"] = newMySudokuOption(true, "Automatic note removal", nil)
	result["doubleclicknotes"] = newMySudokuOption(false, "Double click toggles note mode", nil)
	result["highlighterrors"] = newMySudokuOption(true, "Highlight conflicting cells", nil)
	result["backgroundstyle"] = newMySudokuOption("default", "Background style", []string{
		"default", "rainbow", "flow",
	})
	return result
}

// Figure out whether a user exists
func (mctx *MakaiContext) sudokuUserExists(name string) (bool, error) {
	var count int64
	err := mctx.sudokuDb.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", name).Scan(&count)
	return count > 0, err
}

// Add sudoku user. checks if username exists, etc.
func (mctx *MakaiContext) RegisterSudokuUser(username string, password string) (int64, error) {
	if !mctx.sudokuUsernameRegex.Match([]byte(username)) {
		return 0, fmt.Errorf("Username malformed!")
	}
	exists, err := mctx.sudokuUserExists(username)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, fmt.Errorf("Username not available!")
	}
	hashraw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	hash := base64.StdEncoding.EncodeToString(hashraw)
	result, err := mctx.sudokuDb.Exec(
		"INSERT INTO users(username, password, admin, created) VALUES (?,?,?,?)",
		username, hash, false, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (mctx *MakaiContext) LoginSudokuUser(username string, password string) (string, error) {
	var passhash string
	var uid int64
	err := mctx.sudokuDb.QueryRow("SELECT password,uid FROM users WHERE username = ?", username).Scan(&passhash, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", &utils.ExpectedError{Message: "User not found!"}
		} else {
			return "", err
		}
	}
	rawhash, err := base64.StdEncoding.DecodeString(passhash)
	if err != nil {
		return "", err
	}
	err = bcrypt.CompareHashAndPassword(rawhash, []byte(password))
	if err != nil {
		log.Printf("Error logging in (password): %s", err)
		return "", &utils.ExpectedError{Message: "Password failure!"}
	}
	session := SudokuUserSession{UserId: uid}
	token, err := jwt.Sign(jwt.HS256, []byte(mctx.config.SudokuSecretKey), session, jwt.MaxAge(time.Duration(mctx.config.SudokuCookieExpire)))
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func (mctx *MakaiContext) GetSudokuSession(token string) (*SudokuUserSession, error) {
	//log.Printf("JWT Token to decode: %s", token)
	verified, err := jwt.Verify(jwt.HS256, []byte(mctx.config.SudokuSecretKey), []byte(token))
	if err != nil {
		return nil, err
	}
	var session SudokuUserSession
	err = verified.Claims(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
	//return mctx.GetSudokuUserById(session.UserId)
}

func (mctx *MakaiContext) GetSudokuUserById(id int64) (*SDBUser, error) {
	result := SDBUser{}
	err := mctx.sudokuDb.Get(&result, "SELECT * FROM users WHERE uid = ?", id)
	return &result, err
}

func (mctx *MakaiContext) GetPuzzleSets(uid int64) ([]PuzzleSetAggregate, error) {
	result := make([]PuzzleSetAggregate, 0)
	err := mctx.sudokuDb.Select(&result,
		"select puzzleset, uid, public, count(*) as count from puzzles where uid = ? or public=1 group by puzzleset",
		uid)
	return result, err
}

func (mctx *MakaiContext) UpdateUserSettings(uid int64, settings string) error {
	_, err := mctx.sudokuDb.Exec("UPDATE users SET settings = ? WHERE uid = ?", settings, uid)
	return err
}

// Convert a sudoku db user to a returnable sudoku user.
func (user *SDBUser) ToUser(loggedin bool) SudokuUser {
	result := SudokuUser{
		Uid:      user.UID,
		Username: user.Username,
		Admin:    user.Admin,
		Exists:   true,
		LoggedIn: loggedin,
		Options:  getDefaultSudokuOptions(),
	}
	rawSettings := make(map[string]any)
	err := json.Unmarshal([]byte(user.SettingsJson), &rawSettings)
	if err != nil {
		// This skips user settings errors. So a user can't "brick" themselves
		// if they send some garbage as settings; everything will just become default.
		log.Printf("Bad user settings for '%s': %s", user.Username, err)
	} else {
		for k, v := range rawSettings {
			dv, ok := result.Options[k]
			if ok {
				dv.Value = v
			} else {
				log.Printf("WARN: found unknown setting %s", k)
			}
		}
	}
	// Now that the values are all set, try to set the json options. Note that
	// if the settings change, you will NEED to refresh this!!
	result.RefreshJsonSettings()
	return result
}

func (user *SudokuUser) RefreshJsonSettings() error {
	result, err := json.Marshal(user.Options)
	if err != nil {
		return err
	}
	user.JsonOptions = string(result)
	return nil
}

func (mctx *MakaiContext) GetPuzzlesetData(puzzleset string, uid int64) ([]QueryByPuzzleset, error) {
	result := make([]QueryByPuzzleset, 0)
	err := mctx.sudokuDb.Select(&result,
		"SELECT p.pid, (c.cid IS NOT NULL) as completed, (i.ipid IS NOT NULL) as paused, c.completed as completedOn, i.paused as pausedOn "+
			"FROM puzzles p LEFT JOIN "+
			"completions c ON c.pid=p.pid LEFT JOIN "+
			"inprogress i ON i.pid=p.pid "+
			"WHERE puzzleset=? AND (c.uid=? OR c.uid IS NULL) AND "+
			"(i.uid=? OR i.uid IS NULL) "+
			"GROUP BY p.pid ORDER BY p.pid ",
		puzzleset, uid, uid,
	)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Number = i + 1
	}
	return result, nil
}

func (mctx *MakaiContext) GetPuzzle(pid int64, uid int64) (*QueryByPid, error) {
	var result QueryByPid
	err := mctx.sudokuDb.Get(&result,
		"SELECT p.*, COALESCE(i.puzzle,'') as playersolution, COALESCE(i.seconds,0) as seconds FROM puzzles p LEFT JOIN "+
			"inprogress i ON p.pid=i.pid WHERE p.pid=? AND "+
			"(i.uid=? OR i.uid IS NULL)",
		pid, uid,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func deleteSudokuProgressRaw(pid int64, uid int64, db utils.DbLike) error {
	_, err := db.Exec("DELETE FROM inprogress WHERE pid = ? AND uid = ?", pid, uid)
	if err != nil {
		return err
	}
	// affected, err := result.RowsAffected()
	// if err != nil {
	// 	return err
	// }
	// if affected < 1 {
	// 	return fmt.Errorf("Couldn't find progress to delete")
	// }
	return nil
}

func (mctx *MakaiContext) DeleteSudokuProgress(pid int64, uid int64) error {
	_, err := mctx.GetPuzzle(pid, uid)
	if err != nil {
		return err
	}
	tx, err := mctx.sudokuDb.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteSudokuProgressRaw(pid, uid, tx)
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

// Set user progress for given puzzle. Will check for completion; returns true if solved
func (mctx *MakaiContext) UpdateSudokuProgress(pid int64, uid int64, data string, seconds int) (bool, error) {
	if seconds == 0 {
		return false, fmt.Errorf("Must report seconds taken on puzzle so far!")
	}
	var sdata SudokuSaveData
	err := json.Unmarshal([]byte(data), &sdata)
	if err != nil {
		return false, err
	}
	puzzle, err := mctx.GetPuzzle(pid, uid)
	if err != nil {
		return false, err
	}
	tx, err := mctx.sudokuDb.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	err = deleteSudokuProgressRaw(pid, uid, tx)
	if err != nil {
		return false, err
	}
	//
	solved := false
	if sdata.Puzzle == puzzle.Solution {
		_, err := tx.Exec(
			"INSERT INTO completions(pid, seconds, uid, completed) VALUES (?,?,?,?)",
			pid, seconds, uid, time.Now(),
		)
		if err != nil {
			return false, err
		}
		solved = true
	} else {
		_, err := tx.Exec(
			"INSERT INTO inprogress(pid, seconds, uid, puzzle, paused) VALUES (?,?,?,?,?)",
			pid, seconds, uid, data, time.Now(),
		)
		if err != nil {
			return false, err
		}
	}
	tx.Commit()
	return solved, nil
}
