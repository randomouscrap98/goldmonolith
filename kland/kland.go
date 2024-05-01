package kland

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	//"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version         = "0.1.0"
	AdminIdKey      = "adminId"
	IsAdminKey      = "isAdmin"
	PostStyleKey    = "postStyle"
	OrphanedPrepend = "Internal_OrphanedImages"
	LongCookie      = 365 * 24 * 60 * 60
)

type KlandContext struct {
	config    *Config
	templates *template.Template
}

func NewKlandContext(config *Config) (*KlandContext, error) {
	// MUST have database exist and in good standing...
	dir, _ := filepath.Split(config.DatabasePath)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return nil, err
	}
	err = CreateTables(config)
	if err != nil {
		return nil, err
	}
	err = utils.VerifyVersionedDb(config, DatabaseVersion)
	if err != nil {
		return nil, err
	}
	// MUST have image folder existing...
	err = os.MkdirAll(config.ImagePath, 0750)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(config.TextPath, 0750)
	if err != nil {
		return nil, err
	}
	// For kland, we initialize the templates first because we don't really need
	// hot reloading (also it's just better for performance... though memory usage...
	templates, err := template.New("alltemplates").Funcs(template.FuncMap{
		"RawHtml": func(c string) template.HTML { return template.HTML(c) },
	}).ParseGlob(filepath.Join(config.TemplatePath, "*.tmpl"))

	if err != nil {
		return nil, err
	}

	// Now we're good to go
	return &KlandContext{
		config:    config,
		templates: templates,
	}, nil
}

func (wc *KlandContext) RunBackground(cancel context.Context, wg *sync.WaitGroup) {
	// A stub, do nothing. But you HAVE to exit the wait group!!
	log.Printf("No background tasks for kland")
	wg.Done()
}

func (kctx *KlandContext) GetDefaultData(r *http.Request) map[string]any {
	admincookie, err := r.Cookie(AdminIdKey)
	thisadminid := ""
	if err == nil {
		thisadminid = admincookie.Value
	}
	stylecookie, err := r.Cookie(PostStyleKey)
	style := ""
	if err == nil {
		style = stylecookie.Value
	}
	rinfo := utils.GetRuntimeInfo()
	result := make(map[string]any)
	result["root"] = kctx.config.RootPath
	result["appversion"] = Version
	result[AdminIdKey] = thisadminid
	result[IsAdminKey] = thisadminid == kctx.config.AdminId
	result[PostStyleKey] = style
	result["runtimeInfo"] = rinfo
	result["requestUri"] = r.URL.RequestURI()
	return result
}

func (kctx *KlandContext) runTemplate(name string, w http.ResponseWriter, data any) {
	err := kctx.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("ERROR: can't load template: %s", err)
		http.Error(w, "Template load error (internal server error!)", http.StatusInternalServerError)
	}
}

func reportDbError(err error, w http.ResponseWriter) {
	log.Printf("ERROR OPENING DB: %s", err)
	http.Error(w, "Error opening database", http.StatusInternalServerError)
}

func (kctx *KlandContext) GetHandler() (http.Handler, error) {
	r := chi.NewRouter()

	// Should probably limit the reads...
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(kctx.config.VisitPerInterval, time.Duration(kctx.config.VisitLimitInterval)))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			db, err := kctx.config.OpenDb()
			if err != nil {
				reportDbError(err, w)
				return
			}
			defer db.Close()
			// Need to get threads from db, is it really ALL of them? Yeesh...
			threads, err := GetAllThreads(db)
			if err != nil {
				log.Printf("ERROR RETRIEVING THREADS: %s", err)
				http.Error(w, "Error retrieving threads", http.StatusInternalServerError)
				return
			}
			threadViews := make([]ThreadView, len(threads))
			for i := range threads {
				threadViews[i] = ConvertThread(threads[i], kctx.config)
			}
			data := kctx.GetDefaultData(r)
			data["threads"] = threadViews
			kctx.runTemplate("index.tmpl", w, data)
		})

		r.Get("/thread/{id}", func(w http.ResponseWriter, r *http.Request) {
			db, err := kctx.config.OpenDb()
			if err != nil {
				reportDbError(err, w)
				return
			}
			defer db.Close()
			data := kctx.GetDefaultData(r)
			idraw := chi.URLParam(r, "id")
			tid, err := strconv.ParseInt(idraw, 10, 64)
			if err != nil {
				http.Error(w, "Bad file ID format", http.StatusBadRequest)
				return
			}
			threads, err := GetThreadById(db, []int64{tid})
			if err != nil {
				log.Printf("ERROR RETRIEVING THREADS: %s", err)
				http.Error(w, "Error retrieving thread", http.StatusInternalServerError)
				return
			}
			if len(threads) != 1 {
				http.Error(w, "Thread not found", http.StatusNotFound)
				return
			}
			posts, err := GetPostsInThread(db, tid)
			if err != nil {
				log.Printf("ERROR RETRIEVING POSTS: %s", err)
				http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
				return
			}
			postViews := make([]PostView, len(posts))
			for i := range posts {
				postViews[i] = ConvertPost(posts[i], kctx.config)
			}
			data["thread"] = ConvertThread(threads[0], kctx.config)
			data["posts"] = postViews
			kctx.runTemplate("thread.tmpl", w, data)
		})
	})

	// Upload endpoints, need extra limiting
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(kctx.config.UploadPerInterval, time.Duration(kctx.config.UploadLimitInterval)))

		r.Post("/admin", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "(Admin action): Kland is limping along in readonly mode", http.StatusTeapot)
		})
		r.Post("/submitpost", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "(Regular post): Kland is limping along in readonly mode", http.StatusTeapot)
		})
		r.Post("/uploadtext", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "(Animation uploader): Kland is limping along in readonly mode", http.StatusTeapot)
		})

		r.Post("/settings", func(w http.ResponseWriter, r *http.Request) {
			// Apparently, we don't support the post styling anymore. Sad... but we
			// keep this functionality here just in case? I don't know why...
			q := r.URL.Query()
			adminid := q.Get("adminid")
			poststyle := q.Get("poststyle")
			redirect := q.Get("redirect")
			if redirect == "" {
				redirect = kctx.config.RootPath
			}
			handleSetting := func(name string, value string) {
				if value == "" {
					utils.DeleteCookie(name, w)
				} else {
					http.SetCookie(w, &http.Cookie{
						Name:   name,
						Value:  value,
						MaxAge: LongCookie,
					})
				}
			}
			handleSetting(AdminIdKey, adminid)
			handleSetting(PostStyleKey, poststyle)
			http.Redirect(w, r, redirect, http.StatusSeeOther)
		})

		r.Post("/uploadimage", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("."))
		})
	})

	// --- Static files -----
	var err error
	err = utils.FileServer(r, "/i/", kctx.config.ImagePath)
	if err != nil {
		return nil, err
	}
	err = utils.FileServer(r, "/a", kctx.config.TextPath)
	if err != nil {
		return nil, err
	}
	err = utils.FileServer(r, "/", kctx.config.StaticFilePath)
	if err != nil {
		return nil, err
	}
	return r, nil
}
