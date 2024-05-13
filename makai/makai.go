package makai

import (
	//"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version = "2.0.0"
)

func (mctx *MakaiContext) GetHandler() (http.Handler, error) {
	r := chi.NewRouter()
	var err error

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := mctx.GetDefaultData(r)
		mctx.RunTemplate("index.tmpl", w, data)
	})

	r.Get("/draw/", func(w http.ResponseWriter, r *http.Request) {
		data := mctx.GetDefaultData(r)
		data["nobg"] = r.URL.Query().Has("nobg")
		mctx.RunTemplate("draw_index.tmpl", w, data)
	})

	r.Get("/animator/", func(w http.ResponseWriter, r *http.Request) {
		data := mctx.GetDefaultData(r)
		data["oroot"] = mctx.config.RootPath + "/animator"
		mctx.RunTemplate("offlineanimator_index.tmpl", w, data)
	})

	r.Get("/tinycomputer/", func(w http.ResponseWriter, r *http.Request) {
		data := mctx.GetDefaultData(r)
		data["oroot"] = mctx.config.RootPath + "/tinycomputer"
		mctx.RunTemplate("tinycomputer_index.tmpl", w, data)
	})

	r.Get("/sudoku/", func(w http.ResponseWriter, r *http.Request) {
		mctx.RenderSudoku("game", w, r)
	})

	r.Get("/sudoku/bgtest/", func(w http.ResponseWriter, r *http.Request) {
		mctx.RenderSudoku("bgtest", w, r)
	})

	r.Get("/sudoku/convert/", func(w http.ResponseWriter, r *http.Request) {
		mctx.RenderSudoku("convert", w, r)
	})

	// These are endpoints which have a heavy limiter set
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(mctx.config.HeavyLimitCount, time.Duration(mctx.config.HeavyLimitInterval)))

		r.Get("/chatlog/", func(w http.ResponseWriter, r *http.Request) {
			mctx.WebSearchChatlogs(w, r)
		})

		r.Get("/draw/manager/", func(w http.ResponseWriter, r *http.Request) {
			mctx.WebDrawManager(w, r.URL.Query())
		})

		/*r.Post("/sudoku/login/", func(w http.ResponseWriter, r *http.Request) {
			var result QueryObject
			var query SudokuLoginQuery
			r.ParseForm()
			err := mctx.decoder.Decode(&query, r.Form)
			if err != nil {
				result = queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err))
			} else if query.Logout {
				utils.DeleteCookie(SudokuCookie, w)
				result = queryFromResult(true)
			} else if query.Username != "" && query.Password != "" {
				if query.Password2 != "" { // User registration
					if query.Password != query.Password2 {
						result = queryFromErrors("Passwords don't match!")
					} else {
						id, err := mctx.RegisterSudokuUser(query.Username, query.Password)
						if err != nil {
							log.Printf("User registration failed: %s", err)
							result = queryFromErrors(fmt.Sprintf("Registration failed: %s", err))
						} else {
							log.Printf("Sudoku user '%s' registered (uid %d)", query.Username, id)
							result = mctx.sudokuLogin(query.Username, query.Password, w)
						}
					}
				} else {
					result = mctx.sudokuLogin(query.Username, query.Password, w)
				}
			} else {
				result = queryFromErrors("Must provide username and password at least! Or logout!")
			}

			utils.RespondJson(result, w, nil)
		})*/

		r.Post("/draw/manager/", func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(int64(mctx.config.MaxFormMemory))
			if err != nil {
				log.Printf("Error parsing form initially: %s", err)
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
				return
			}
			mctx.WebDrawManager(w, r.PostForm)
		})
	})

	// Static file path
	err = utils.FileServer(r, "/", mctx.config.StaticFilePath, true)
	if err != nil {
		return nil, err
	}
	return r, nil
}
