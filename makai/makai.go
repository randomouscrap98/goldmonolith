package makai

import (
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

	// These are endpoints which have a heavy limiter set
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(mctx.config.HeavyLimitCount, time.Duration(mctx.config.HeavyLimitInterval)))

		r.Get("/chatlog/", func(w http.ResponseWriter, r *http.Request) {
			mctx.WebSearchChatlogs(w, r)
		})

		r.Get("/draw/manager/", func(w http.ResponseWriter, r *http.Request) {
			mctx.WebDrawManager(w, r.URL.Query())
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

	var err error

	// Static file path
	err = utils.FileServer(r, "/", mctx.config.StaticFilePath, true)
	if err != nil {
		return nil, err
	}
	return r, nil
}
