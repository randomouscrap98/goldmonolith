package makai

import (
	"encoding/base64"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// const (
//   SudokuUserSession
// )

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

// Compare a password with a hash generated from passwordHash
func passwordVerify(password string, hash string) error {
	rawhash, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword(rawhash, []byte(password))
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

	result, err := mctx.sudokuDb.Exec("INSERT INTO users(username, password, admin) VALUES (?,?,?)",
		username, hash, false)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (mctx *MakaiContext) GetSudokuUserById(id int64) (*SDBUser, error) {
	result := SDBUser{}
	err := mctx.sudokuDb.Get(&result, "SELECT * FROM users WHERE uid = ?", id)
	return &result, err
}
