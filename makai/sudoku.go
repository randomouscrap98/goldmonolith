package makai

import (
	"encoding/base64"
	"net/http"

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
