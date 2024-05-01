package kland

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	templates, err := template.New("alltemplates").Funcs(template.FuncMap{}).ParseGlob(filepath.Join(config.TemplatePath, "*.tmpl"))

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
	result["appversion"] = Version
	result[AdminIdKey] = thisadminid
	result[IsAdminKey] = thisadminid == kctx.config.AdminId
	result[PostStyleKey] = style
	result["runtimeInfo"] = rinfo
	return result
}

// func (kctx *KlandContext) InitializeTemplate(files []string) (*template.Template, error) {
//   filepaths := make([]string, len(files))
//   for i := range files {
//     filepaths[i] = filepath.Join(kctx.config.TemplatePath, files[i])
//   }
//   return template.New(files[0]).Funcs(template.FuncMap{
//   }).ParseFiles(filepaths...)
// }
//
// func (kctx *KlandContext) InitializeIndexTemplate() (*template.Template, error) {
//   return kctx.InitializeTemplate([]string {
//     "index.tmpl",
//     "header.tmpl",
//     "footer.tmpl",
//   })
// }

func (kctx *KlandContext) GetHandler() (http.Handler, error) {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := kctx.GetDefaultData(r)
		// Need to get threads from db, is it really ALL of them? Yeesh...
		threads, err := GetAllThreads(kctx.config)
		if err != nil {
			http.Error(w, "Error retrieving threads", http.StatusInternalServerError)
			return
		}
		threadViews := make([]ThreadView, len(threads))
		for i := range threads {
			threadViews[i] = ConvertThread(threads[i])
		}
		data["threads"] = threadViews
	})

	// Upload endpoints, need extra limiting
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(kctx.config.UploadPerInterval, time.Duration(kctx.config.UploadLimitInterval)))

		r.Post("/uploadtext", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("."))
		})
		r.Post("/uploadimage", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("."))
		})
	})

	// --- Static files -----
	err := utils.FileServer(r, "/", kctx.config.StaticFilePath)
	if err != nil {
		return nil, err
	}
	return r, nil
}
