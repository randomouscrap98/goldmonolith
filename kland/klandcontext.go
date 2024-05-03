package kland

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	//"time"

	"github.com/gorilla/schema"

	"github.com/randomouscrap98/goldmonolith/utils"
)

type KlandContext struct {
	config        *Config
	decoder       *schema.Decoder
	templates     *template.Template
	tinsmu        sync.Mutex
	pinsmu        sync.Mutex
	rawImageRegex *regexp.Regexp
}

func NewKlandContext(config *Config) (*KlandContext, error) {
	// MUST have database exist and in good standing...
	dir, _ := filepath.Split(config.DatabasePath)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return nil, err
	}
	db, err := config.OpenDb()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = CreateTables(db)
	if err != nil {
		return nil, err
	}
	err = utils.VerifyVersionedDb(db, DatabaseVersion)
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
		decoder:   schema.NewDecoder(),
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

func (kctx *KlandContext) RunTemplate(name string, w http.ResponseWriter, data any) {
	err := kctx.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("ERROR: can't load template: %s", err)
		http.Error(w, "Template load error (internal server error!)", http.StatusInternalServerError)
	}
}

func (kctx *KlandContext) CheckThreadsConvert(threads []Thread, err error, w http.ResponseWriter) []ThreadView {
	if err != nil {
		log.Printf("ERROR RETRIEVING THREADS: %s", err)
		http.Error(w, "Error retrieving threads", http.StatusInternalServerError)
		return nil
	}
	threadViews := make([]ThreadView, len(threads))
	for i := range threads {
		threadViews[i] = ConvertThread(threads[i], kctx.config)
	}
	return threadViews
}

func (kctx *KlandContext) CheckPostsConvert(posts []Post, err error, w http.ResponseWriter) []PostView {
	if err != nil {
		log.Printf("ERROR RETRIEVING POSTS: %s", err)
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return nil
	}
	postViews := make([]PostView, len(posts))
	for i := range posts {
		postViews[i] = ConvertPost(posts[i], kctx.config)
	}
	return postViews
}

type GetImageQuery struct {
	Bucket string `schema:"bucket"`
	AsJSON bool   `schema:"asJSON"`
	Page   int    `schema:"page"`
	IPP    int    `schema:"ipp"`
	View   string `schema:"view"`
}

func (iquery *GetImageQuery) IntoData(data map[string]any) {
	// Unfortunately, because we're returning json, we HAVE to do this silliness
	data["bucket"] = iquery.Bucket
	data["ipp"] = iquery.IPP
	data["view"] = iquery.View
	data["page"] = iquery.Page
	data["nextPage"] = iquery.Page + 1
	if iquery.Page > 1 {
		data["previousPage"] = iquery.Page - 1
	}
}

func (kctx *KlandContext) ParseImageQuery(r *http.Request) (GetImageQuery, error) {
	params := r.URL.Query()
	iquery := GetImageQuery{}
	err := kctx.decoder.Decode(&iquery, params)
	if err != nil {
		return iquery, err
	}
	// This uses the cookie IF it exists, otherwise it uses the query value
	iquery.IPP = utils.GetCookieOrDefault("ipp", r, iquery.IPP, func(s string) (int, error) {
		return strconv.Atoi(s)
	})
	if iquery.Page < 1 {
		iquery.Page = 1
	}
	if iquery.IPP <= 0 {
		iquery.IPP = DefaultIpp
	}
	return iquery, nil
}

func bucketSubject(bucket string) string {
	if bucket == "" {
		return OrphanedPrepend
	} else {
		return fmt.Sprintf("%s_%s", OrphanedPrepend, bucket)
	}
}

// Either retrieve the existing bucket thread, or create a new one. It will always
// have a valid hash after this call, even if it previously did not.
func (kctx *KlandContext) GetOrCreateBucketThread(db *sql.DB, bucket string) (*Thread, error) {
	subject := bucketSubject(bucket)
	threads, err := GetThreadsByField(db, "subject", subject)
	if err != nil {
		return nil, err
	}
	// Probably safer to just lock here... not too bad
	kctx.tinsmu.Lock()
	defer kctx.tinsmu.Unlock()
	if len(threads) > 0 {
		// if the thread exists, just check for hash update.
		thread := threads[0]
		if utils.IsNilOrEmpty(thread.hash) {
			log.Printf("Thread %s(%d) doesn't have a hash; generating", thread.subject, thread.tid)
			return UpdateThreadHash(db, thread.tid)
		} else {
			return &thread, nil
		}
	} else {
		// The thread needs to be created
		return InsertBucketThread(db, subject)
	}
}
