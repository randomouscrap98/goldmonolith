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

func (mctx *MakaiContext) GetSudokuSession(token string) (*SDBUser, error) {
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
	return mctx.GetSudokuUserById(session.UserId)
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

// private Task<IEnumerable<PuzzleSetAggregate>> GetPuzzleSets(int uid) => SimpleDbTask(con =>
//     con.QueryAsync<PuzzleSetAggregate>(
//         "select puzzleset, uid, public, count(*) as count from puzzles where uid = @uid or public=1 group by puzzleset",
//         new {uid = uid})
// );

// Convert a sudoku db user to a returnable sudoku user.
func (user *SDBUser) ToUser(loggedin bool) (SudokuUser, error) {
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
		return result, err
	}
	for k, v := range rawSettings {
		dv, ok := result.Options[k]
		if ok {
			dv.Value = v
		} else {
			log.Printf("WARN: found unknown setting %s", k)
		}
	}
	// Now that the values are all set, try to set the json options. Note that
	// if the settings change, you will NEED to refresh this!!
	result.RefreshJsonSettings()
	return result, nil
}

func (user *SudokuUser) RefreshJsonSettings() error {
	result, err := json.Marshal(user.Options)
	if err != nil {
		return err
	}
	user.JsonOptions = string(result)
	return nil
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
