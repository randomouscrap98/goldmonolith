package utils

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"path/filepath"
	"strings"
)

// Taken from: https://github.com/go-chi/chi/blob/master/_examples/fileserver/main.go
// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServerRaw(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	// There's a bug here: what if path is empty?
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func FileServer(r chi.Router, path string, local string) error {
	staticPath, err := filepath.Abs(local)
	if err != nil {
		panic(err)
	}
	FileServerRaw(r, path, http.Dir(staticPath))
	return nil
}
