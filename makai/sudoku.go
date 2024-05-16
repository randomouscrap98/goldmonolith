package makai

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kataras/jwt"
	"golang.org/x/crypto/bcrypt"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	SudokuCookie = "makai_sudoku_session"
)

type SudokuLoginQuery struct {
	Username  string `schema:"username"`
	Password  string `schema:"password"`
	Password2 string `schema:"password2"`
	Logout    bool   `schema:"logout"`
}

type SudokuUserSession struct {
	UserId int64 `json:"uid"`
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

func queryFromError(err error) QueryObject {
	_, ok := err.(*utils.ExpectedError)
	if ok {
		return queryFromErrors(err.Error())
	} else {
		log.Printf("INTERNAL SERVER ERROR: %s", err)
		return queryFromErrors("Internal server error")
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

func (mctx *MakaiContext) GetSudokuSession(token string) (*SDBUser, error) {
	// cookie, err := r.Cookie(SudokuCookie)
	// if err != nil {
	// 	return nil, err
	// }
	verified, err := jwt.Verify(jwt.HS256, []byte(mctx.config.SudokuSecretKey), []byte(token))
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

func (mctx *MakaiContext) GetSudokuUserById(id int64) (*SDBUser, error) {
	result := SDBUser{}
	err := mctx.sudokuDb.Get(&result, "SELECT * FROM users WHERE uid = ?", id)
	return &result, err
}

// func (mctx *MakaiContext) GetSudokuUserByName(name string) (*SDBUser, error) {
// 	result := SDBUser{}
// 	err := mctx.sudokuDb.Get(&result, "SELECT * FROM users WHERE username = ?", name)
// 	return &result, err
// }

// // Fill the settings fields and stuff
// func (mctx *MakaiContext) FillSudokuUser(user *SudokuUser) error {
//   rows, err := mctx.sudokuDb.Query("SELECT * FROM settings WHERE uid = ?", user.UID)
//   if err!= nil {
//     return err
//   }
//   defer rows.Close()
//   for rows.next() {
//
//   }
//
//   settings := make([]SDBSetting,
//   mctx.Select(
//         var initialResult = await con.QueryAsync<SDBSetting>("select * from settings where uid = @uid", new { uid = uid });
//         return initialResult.ToDictionary(x => x.name, y => JsonConvert.DeserializeObject(y.value));
//
//             result.options = DefaultOptions;
//             var options = await GetRawSettingsForUser(uid);
//
//             foreach (var option in options)
//             {
//                 if(result.options.ContainsKey(option.Key))
//                     result.options[option.Key].value = option.Value;
//             }
// }

// ------------------ WEB STUFF ------------------
