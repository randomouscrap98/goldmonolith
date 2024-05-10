package makai

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/schema"

	"github.com/randomouscrap98/goldmonolith/utils"
)

type MakaiContext struct {
	config              *Config
	decoder             *schema.Decoder
	templates           *template.Template
	drawRegex           *regexp.Regexp
	chatlogIncludeRegex *regexp.Regexp
	created             time.Time
	drawDataMu          sync.Mutex
}

func NewMakaiContext(config *Config) (*MakaiContext, error) {
	// MUST have drawings path exist
	err := os.MkdirAll(config.DrawingsPath, 0750)
	if err != nil {
		return nil, err
	}
	drawRegex, err := regexp.Compile(config.DrawSafetyRegex)
	if err != nil {
		return nil, err
	}
	chatlogIncludeRegex, err := regexp.Compile(config.ChatlogIncludeRegex)
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
		config:              config,
		templates:           templates,
		decoder:             schema.NewDecoder(),
		drawRegex:           drawRegex,
		chatlogIncludeRegex: chatlogIncludeRegex,
		created:             time.Now(),
	}, nil
}

func (wc *MakaiContext) RunBackground(cancel context.Context, wg *sync.WaitGroup) {
	// A stub, do nothing. But you HAVE to exit the wait group!!
	log.Printf("No background tasks for makai (yet)")
	wg.Done()
}

func (wc *MakaiContext) GetIdentifier() string {
	return "Makai - " + Version
}

// Retrieve the default data for any page load. Add your additional data to this
// map before rendering
func (kctx *MakaiContext) GetDefaultData(r *http.Request) map[string]any {
	rinfo := utils.GetRuntimeInfo()
	result := make(map[string]any)
	result["root"] = template.URL(kctx.config.RootPath)
	result["appversion"] = Version
	result["runtimeInfo"] = rinfo
	result["requestUri"] = r.URL.RequestURI()
	result["cachebust"] = kctx.created.Format(time.RFC3339)
	result["klandurl"] = template.URL(kctx.config.KlandUrl)
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
