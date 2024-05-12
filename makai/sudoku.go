package makai

import (
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kataras/jwt"
	"golang.org/x/crypto/bcrypt"
	//"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	SudokuCookie = "makai_sudoku_session"
)

func (mctx *MakaiContext) RenderSudoku(subtemplate string, w http.ResponseWriter, r *http.Request) {
	data := mctx.GetDefaultData(r)
	data["oroot"] = mctx.config.RootPath + "/sudoku"
	data["template_"+subtemplate] = true
	data["debug"] = r.URL.Query().Has("debug")
	data["puzzleSets"] = "" // Some serialized thing...
	_, err := os.Stat(mctx.config.SudokuDbPath)
	data["dbexists"] = err == nil
	mctx.RunTemplate("sudoku_index.tmpl", w, data)
}

// Generate a hash string from the given password.
func passwordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

// Try a login
func (mctx *MakaiContext) sudokuPasswordVerify(username string, password string) (bool, error) {
	var passhash string
	err := mctx.sudokuDb.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&passhash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		} else {
			return true, err
		}
	}
	rawhash, err := base64.StdEncoding.DecodeString(passhash)
	if err != nil {
		return true, err
	}
	return true, bcrypt.CompareHashAndPassword(rawhash, []byte(password))
}

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

func (mctx *MakaiContext) RegisterSudokuUser(username string, password string) (int64, error) {
	hash, err := passwordHash(password)
	if err != nil {
		return 0, err
	}
	result, err := mctx.sudokuDb.Exec(
		"INSERT INTO users(username, password, admin) VALUES (?,?,?)",
		username, hash, false,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (mctx *MakaiContext) GetSudokuUserById(id int64) (*SudokuUser, error) {
	result := SudokuUser{}
	err := mctx.sudokuDb.Get(&result, "SELECT * FROM users WHERE uid = ?", id)
	return &result, err
}

func (mctx *MakaiContext) GetSudokuUserByName(name string) (*SudokuUser, error) {
	result := SudokuUser{}
	err := mctx.sudokuDb.Get(&result, "SELECT * FROM users WHERE username = ?", name)
	return &result, err
}

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
	}
}

type SudokuLoginQuery struct {
	Username  string `schema:"username"`
	Password  string `schema:"password"`
	Password2 string `schema:"password2"`
	Logout    bool   `schema:"logout"`
}

type SudokuUserSession struct {
	UserId int64 `json:"uid"`
}

// ------------------ WEB STUFF ------------------

func (mctx *MakaiContext) sudokuLogin(username string, password string, w http.ResponseWriter) QueryObject {
	var passhash string
	var uid int64
	err := mctx.sudokuDb.QueryRow("SELECT password,uid FROM users WHERE username = ?", username).Scan(&passhash, &uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return queryFromErrors("User not found!")
		} else {
			log.Printf("Error logging in (db query): %s", err)
			return queryFromErrors("Internal server error")
		}
	}
	rawhash, err := base64.StdEncoding.DecodeString(passhash)
	if err != nil {
		log.Printf("Error logging in (db query): %s", err)
		return queryFromErrors("Internal server error")
	}
	err = bcrypt.CompareHashAndPassword(rawhash, []byte(password))
	if err != nil {
		log.Printf("Error logging in (password): %s", err)
		return queryFromErrors("Password failure!")
	}
	session := SudokuUserSession{UserId: uid}
	token, err := jwt.Sign(jwt.HS256, []byte(mctx.config.SudokuSecretKey), session, jwt.MaxAge(time.Duration(mctx.config.SudokuCookieExpire)))
	if err != nil {
		log.Printf("Error logging in (sign token): %s", err)
		return queryFromErrors("Internal server error (token)")
	}
	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:   SudokuCookie,
		Value:  string(token),
		MaxAge: int(time.Duration(mctx.config.SudokuCookieExpire).Seconds()),
	})
	return queryFromResult(true)
}

func (mctx *MakaiContext) sudokuGetCurrentUser(r *http.Request) (*SudokuUser, error) {
	cookie, err := r.Cookie(SudokuCookie)
	if err != nil {
		return nil, err
	}
	verified, err := jwt.Verify(jwt.HS256, []byte(mctx.config.SudokuSecretKey), []byte(cookie.Value))
	if err != nil {
		return nil, err
	}
	var session SudokuUserSession
	err = verified.Claims(&session)
	if err != nil {
		return nil, err
	}
	return mctx.GetSudokuUserById(session.UserId)
}
