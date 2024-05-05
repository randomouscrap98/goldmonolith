package makai

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version = "0.1.0"
)

func (kctx *MakaiContext) GetHandler() (http.Handler, error) {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := kctx.GetDefaultData(r)
		kctx.RunTemplate("index.tmpl", w, data)
	})

	r.Get("/draw/", func(w http.ResponseWriter, r *http.Request) {
		data := kctx.GetDefaultData(r)
		data["nobg"] = r.URL.Query().Has("nobg")
		kctx.RunTemplate("draw_index.tmpl", w, data)
	})

	r.Get("/animator/", func(w http.ResponseWriter, r *http.Request) {
		data := kctx.GetDefaultData(r)
		data["oroot"] = kctx.config.RootPath + "/animator"
		kctx.RunTemplate("offlineanimator_index.tmpl", w, data)
	})

	r.Get("/tinycomputer/", func(w http.ResponseWriter, r *http.Request) {
		data := kctx.GetDefaultData(r)
		data["oroot"] = kctx.config.RootPath + "/tinycomputer"
		kctx.RunTemplate("tinycomputer_index.tmpl", w, data)
	})

	var err error

	// Static file path
	err = utils.FileServer(r, "/", kctx.config.StaticFilePath)
	if err != nil {
		return nil, err
	}
	return r, nil
}
