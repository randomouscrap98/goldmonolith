package makai

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version = "2.0.1"
)

func (mctx *MakaiContext) GetHandler() (http.Handler, error) {
	r := chi.NewRouter()
	var err error

	// --------------------
	//  Random static only
	// --------------------
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

	// ---------------------
	//    Sudoku
	// ---------------------
	r.Get("/sudoku/", func(w http.ResponseWriter, r *http.Request) {
		mctx.RenderSudoku("game", w, r)
	})

	r.Get("/sudoku/bgtest/", func(w http.ResponseWriter, r *http.Request) {
		mctx.RenderSudoku("bgtest", w, r)
	})

	r.Get("/sudoku/convert/", func(w http.ResponseWriter, r *http.Request) {
		mctx.RenderSudoku("convert", w, r)
	})

	r.Post("/sudoku/sudokuquery", func(w http.ResponseWriter, r *http.Request) {
		result, err := mctx.SudokuQueryWeb(r)
		if err != nil {
			utils.RespondJson(queryFromError(err), w, nil)
		} else {
			utils.RespondJson(queryFromResult(result), w, nil)
		}
	})

	r.Post("/sudoku/puzzlesave", func(w http.ResponseWriter, r *http.Request) {
		result := func(result QueryObject) {
			utils.RespondJson(result, w, nil)
		}
		// Before doing anything, might as well make sure they're logged in...
		uid, err := mctx.GetLoggedInSudokuUid(r)
		if err != nil || uid <= 0 {
			if err != nil {
				log.Printf("Error retrieving logged in sudoku user: %s", err)
			}
			result(queryFromErrors("Must be logged in to save puzzles!"))
			return
		}
		err = r.ParseMultipartForm(int64(mctx.config.MaxFormMemory))
		if err != nil {
			result(queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err)))
			return
		}
		var query SudokuSaveQuery
		err = mctx.decoder.Decode(&query, r.PostForm)
		if err != nil {
			result(queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err)))
			return
		}
		if query.Pid == 0 {
			result(queryFromErrors("Must provide pid"))
			return
		}
		if query.Data != "" {
			solved, err := mctx.UpdateSudokuProgress(query.Pid, uid, query.Data, query.Seconds)
			if err != nil {
				result(queryFromError(err))
			} else {
				if solved {
					result(queryFromResult("completed"))
				} else {
					result(queryFromResult("saved"))
				}
			}
		} else if query.Delete {
			err = mctx.DeleteSudokuProgress(query.Pid, uid)
			if err != nil {
				result(queryFromError(err))
			} else {
				result(queryFromResult(true))
			}
		} else {
			result(queryFromErrors("Invalid parameters! Must provide either data or set delete!"))
		}
	})

	// ----------------------------------------------------
	// These are endpoints which have a heavy limiter set
	// ----------------------------------------------------
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(mctx.config.HeavyLimitCount, time.Duration(mctx.config.HeavyLimitInterval)))

		r.Get("/chatlog/", func(w http.ResponseWriter, r *http.Request) {
			mctx.WebSearchChatlogs(w, r)
		})

		r.Get("/draw/manager/", func(w http.ResponseWriter, r *http.Request) {
			mctx.WebDrawManager(w, r.URL.Query())
		})

		r.Post("/sudoku/settingsave", func(w http.ResponseWriter, r *http.Request) {
			result := func(result QueryObject) {
				utils.RespondJson(result, w, nil)
			}
			// Before doing anything, might as well make sure they're logged in...
			uid, err := mctx.GetLoggedInSudokuUid(r)
			if err != nil || uid <= 0 {
				if err != nil {
					log.Printf("Error retrieving logged in sudoku user: %s", err)
				}
				result(queryFromErrors("Must be logged in to set settings!"))
				return
			}
			err = r.ParseMultipartForm(int64(mctx.config.MaxFormMemory))
			if err != nil {
				result(queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err)))
				return
			}
			settingsraw := r.FormValue("settings")
			if settingsraw == "" {
				result(queryFromErrors("Empty settings"))
				return
			}
			// Don't even need to parse the settings, just update directly. Whatever
			// the user wants... even if it messes up their user? Hm...
			err = mctx.UpdateUserSettings(uid, settingsraw)
			if err != nil {
				result(queryFromErrors(fmt.Sprintf("Couldn't update settings: %s", err)))
				return
			}
			log.Printf("Sudoku user '%d' updated settings", uid)
			result(queryFromResult(true))
		})

		r.Post("/sudoku/login", func(w http.ResponseWriter, r *http.Request) {
			var query SudokuLoginQuery
			result := func(result QueryObject) {
				//log.Printf("Query: %v", query)
				//log.Printf("Login result: %v", result)
				utils.RespondJson(result, w, nil)
			}
			// Multiple codepaths must do this, so like.. whatever
			login := func() {
				token, err := mctx.LoginSudokuUser(query.Username, query.Password)
				if err != nil {
					result(queryFromError(err))
				} else {
					http.SetCookie(w, &http.Cookie{
						Name:   SudokuCookie,
						Value:  token,
						MaxAge: int(time.Duration(mctx.config.SudokuCookieExpire).Seconds()),
					})
					result(queryFromResult(true))
				}
			}
			err := r.ParseMultipartForm(int64(mctx.config.MaxFormMemory))
			if err != nil {
				result(queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err)))
				return
			}
			err = mctx.decoder.Decode(&query, r.PostForm)
			if err != nil {
				result(queryFromErrors(fmt.Sprintf("Couldn't parse form: %s", err)))
				return
			}
			if query.Logout {
				utils.DeleteCookie(SudokuCookie, w)
				result(queryFromResult(true))
				return
			} else if query.Username != "" && query.Password != "" {
				if query.Password2 != "" { // User registration
					if query.Password != query.Password2 {
						result(queryFromErrors("Passwords don't match!"))
						return
					}
					id, err := mctx.RegisterSudokuUser(query.Username, query.Password)
					if err != nil {
						log.Printf("User registration failed: %s", err)
						result(queryFromErrors(fmt.Sprintf("Registration failed: %s", err)))
						return
					}
					log.Printf("Sudoku user '%s' registered (uid %d)", query.Username, id)
					login() // Login after successful registration
				} else { // Actual login
					login()
				}
			} else {
				result(queryFromErrors("Must provide username and password at least! Or logout!"))
				return
			}
		})

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

	// -------------------
	// Static file path
	// -------------------
	err = utils.FileServer(r, "/", mctx.config.StaticFilePath, true)
	if err != nil {
		return nil, err
	}

	return r, nil
}
