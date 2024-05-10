package makai

import (
	"net/http"
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
