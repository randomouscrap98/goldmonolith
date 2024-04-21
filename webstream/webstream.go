package webstream

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func GetHandler() http.Handler {
	r := chi.NewRouter()

	r.Get("/constants", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is the constants endpoint"))
	})

	return r
}
