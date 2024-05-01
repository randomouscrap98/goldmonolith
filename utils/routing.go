package utils

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	DefaultCacheControl = "public, max-age=15552000" // 6 months
)

// Adds a robots.txt that disallows everything to the router. It of course
// is served at root. It might be better to include a robots.txt in the
// static file list to give more control, however...
func AngryRobots(r *chi.Mux) {
	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("User-agent: *\nDisallow: /\n"))
	})
}

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
		w.Header().Set("Cache-Control", DefaultCacheControl)
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
