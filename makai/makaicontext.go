package makai

import (
	"context"
	//"database/sql"
	//"fmt"
	"html/template"
	//"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	//"strconv"
	"sync"
	"time"

	"github.com/gorilla/schema"

	"github.com/randomouscrap98/goldmonolith/utils"
)

type MakaiContext struct {
	config    *Config
	decoder   *schema.Decoder
	templates *template.Template
	//tinsmu    sync.Mutex
	//pinsmu    sync.Mutex
	created time.Time
}

func NewMakaiContext(config *Config) (*MakaiContext, error) {
	// MUST have drawings path exist
	err := os.MkdirAll(config.DrawingsPath, 0750)
	if err != nil {
		return nil, err
	}
	// For makai, we initialize the templates first because we don't really need
	// hot reloading (also it's just better for performance... though memory usage...
	templates, err := template.New("alltemplates").Funcs(template.FuncMap{
		"RawHtml": func(c string) template.HTML { return template.HTML(c) },
	}).ParseGlob(filepath.Join(config.TemplatePath, "*.tmpl"))

	if err != nil {
		return nil, err
	}

	// Now we're good to go
	return &MakaiContext{
		config:    config,
		templates: templates,
		decoder:   schema.NewDecoder(),
		created:   time.Now(),
	}, nil
}

func (wc *MakaiContext) RunBackground(cancel context.Context, wg *sync.WaitGroup) {
	// A stub, do nothing. But you HAVE to exit the wait group!!
	log.Printf("No background tasks for makai (yet)")
	wg.Done()
}

// Retrieve the default data for any page load. Add your additional data to this
// map before rendering
func (kctx *MakaiContext) GetDefaultData(r *http.Request) map[string]any {
	// admincookie, err := r.Cookie(AdminIdKey)
	// thisadminid := ""
	// if err == nil {
	// 	thisadminid = admincookie.Value
	// }
	// stylecookie, err := r.Cookie(PostStyleKey)
	// style := ""
	// if err == nil {
	// 	style = stylecookie.Value
	// }
	rinfo := utils.GetRuntimeInfo()
	result := make(map[string]any)
	result["root"] = kctx.config.RootPath
	result["appversion"] = Version
	//result[AdminIdKey] = thisadminid
	//result[IsAdminKey] = thisadminid == kctx.config.AdminId
	//result[PostStyleKey] = style
	result["runtimeInfo"] = rinfo
	result["requestUri"] = r.URL.RequestURI()
	result["cachebust"] = kctx.created.Format(time.RFC3339)
	//"RawHtml": func(c string) template.HTML { return template.HTML(c) },
	return result
}

// Call this instead of directly accessing templates to do a final render of a page
func (kctx *MakaiContext) RunTemplate(name string, w http.ResponseWriter, data any) {
	err := kctx.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("ERROR: can't load template: %s", err)
		http.Error(w, "Template load error (internal server error!)", http.StatusInternalServerError)
	}
}

// func (kctx *KlandContext) WriteTemp(r io.Reader, w http.ResponseWriter) (*os.File, error) {
// 	err := os.MkdirAll(kctx.config.TempPath, 0700)
// 	if err != nil {
// 		log.Printf("Couldn't create temp folder: %s", err)
// 		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
// 		return nil, err
// 	}
// 	tempfile, err := os.CreateTemp(kctx.config.TempPath, "kland_upload_")
// 	if err != nil {
// 		log.Printf("Couldn't open temp file: %s", err)
// 		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
// 		return nil, err
// 	}
// 	_, err = io.Copy(tempfile, r)
// 	if err != nil {
// 		tempfile.Close()
// 		log.Printf("Couldn't write temp file: %s", err)
// 		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
// 		return nil, err
// 	}
// 	_, err = tempfile.Seek(0, io.SeekStart)
// 	if err != nil {
// 		tempfile.Close()
// 		log.Printf("Couldn't seek temp file: %s", err)
// 		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
// 		return nil, err
// 	}
// 	return tempfile, nil
// }
