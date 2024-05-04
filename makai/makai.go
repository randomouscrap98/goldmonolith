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

	var err error

	// Static file path
	err = utils.FileServer(r, "/", kctx.config.StaticFilePath)
	if err != nil {
		return nil, err
	}
	return r, nil
}
