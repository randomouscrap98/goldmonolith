package kland

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/schema"

	"github.com/randomouscrap98/goldmonolith/utils"
)

type KlandContext struct {
	config    *Config
	decoder   *schema.Decoder
	templates *template.Template
	tinsmu    sync.Mutex
	pinsmu    sync.Mutex
	created   time.Time
}

func NewKlandContext(config *Config) (*KlandContext, error) {
	// MUST have database exist and in good standing...
	dir, _ := filepath.Split(config.DatabasePath())
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
	err = os.MkdirAll(config.ImagePath(), 0750)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(config.TextPath(), 0750)
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
		created:   time.Now(),
	}, nil
}

func (wc *KlandContext) RunBackground(cancel context.Context, wg *sync.WaitGroup) {
	// A stub, do nothing. But you HAVE to exit the wait group!!
	log.Printf("No background tasks for kland")
	wg.Done()
}

// Retrieve the default data for any page load. Add your additional data to this
// map before rendering
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
	result["cachebust"] = kctx.created.Format(time.RFC3339)
	//"RawHtml": func(c string) template.HTML { return template.HTML(c) },
	return result
}

// Call this instead of directly accessing templates to do a final render of a page
func (kctx *KlandContext) RunTemplate(name string, w http.ResponseWriter, data any) {
	err := kctx.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("ERROR: can't load template: %s", err)
		http.Error(w, "Template load error (internal server error!)", http.StatusInternalServerError)
	}
}

// Does what it says on the tin: if you get a result of many threads, it checks the error
// for you, writes results to the response, and converts all the threads to views.
func (kctx *KlandContext) ConvertThreadResult(threads []Thread, err error, w http.ResponseWriter) []ThreadView {
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

// Does what it says on the tin: if you get a result of many posts, it checks the error
// for you, writes results to the response, and converts all the posts to views.
func (kctx *KlandContext) ConvertPostResult(posts []Post, err error, w http.ResponseWriter) []PostView {
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

// Parse image query out of request, requires decoder in context
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
		iquery.IPP = kctx.config.DefaultIPP
	}
	return iquery, nil
}

func (kctx *KlandContext) ParseImageUploadQuery(r *http.Request) UploadImageQuery {
	result := UploadImageQuery{}
	result.raw = r.FormValue("raw")
	result.animation = r.FormValue("animation")
	result.redirect = utils.StringToBool(r.FormValue("redirect"))
	result.short = utils.StringToBool(r.FormValue("shorturl"))
	result.ipaddress = r.Header.Get(kctx.config.IpHeader)
	if result.ipaddress == "" {
		result.ipaddress = "unknown"
	}
	result.bucket = r.FormValue("bucket")
	return result
}

// Either retrieve the existing bucket thread, or create a new one. It will always
// have a valid hash after this call, even if it previously did not.
func (kctx *KlandContext) GetOrCreateBucketThread(db *sql.DB, bucket string) (Thread, error) {
	// NOTE: tried to do naked returns, it was awful, just did it another way
	var thread Thread
	subject := bucketSubject(bucket)
	threads, err := GetThreadsByField(db, "subject", subject)
	if err != nil {
		return thread, err
	}
	// Probably safer to just lock here... not too bad
	kctx.tinsmu.Lock()
	defer kctx.tinsmu.Unlock()
	if len(threads) > 0 {
		// if the thread exists, just check for hash update.
		thread = threads[0]
		if thread.hash != "" {
			return thread, nil // THE most normal return. Nearly all will go through here
		}
		log.Printf("Thread %s(%d) doesn't have a hash; generating", thread.subject, thread.tid)
		_, err = UpdateThreadHash(db, thread.tid)
		if err != nil {
			return thread, err // Bad db
		}
	} else {
		// The thread needs to be created
		thread.tid, thread.hash, err = InsertBucketThread(db, subject)
		if err != nil {
			return thread, err // Bad db
		}
	}
	// Now we must lookup the thread updated thread... not very fun
	threads, err = GetThreadsById(db, []int64{thread.tid})
	if err != nil {
		return thread, err
	}
	if len(threads) < 1 {
		return thread, &utils.NotFoundError{}
	}
	return threads[0], nil
}

func (kctx *KlandContext) GenerateRandomUniqueFilename(extension string) (string, error) {
	if len(extension) > 0 && extension[0] == '.' {
		extension = extension[1:]
	}
	if len(extension) == 0 {
		return "", fmt.Errorf("you must provide an extension")
	}
	folder := kctx.config.ImagePath()
	// Maybe change this to generate more...
	lowerExt := strings.ToLower(extension)
	upperExt := strings.ToUpper(extension)
	// generate a valid file name (one that is not currently used)
	retries := 0
	var name string
	for {
		name = utils.RandomAsciiName(kctx.config.HashBaseChars + retries/kctx.config.HashIncreaseRetries)
		found, err := utils.CheckAnyPathExists([]string{
			filepath.Join(folder, fmt.Sprintf("%s.%s", name, lowerExt)),
			filepath.Join(folder, fmt.Sprintf("%s.%s", name, upperExt)),
		})
		if err != nil {
			return "", err
		}
		// Nothing found, that's good, it's a usable file
		if !found {
			break
		} else {
			log.Printf("Collision: %s (retries: %d)", name, retries)
		}
		retries += 1
	}
	// now we move the file and we're done
	filename := fmt.Sprintf("%s.%s", name, extension)
	return filename, nil
}

func (kctx *KlandContext) WriteTemp(r io.Reader, w http.ResponseWriter) (*os.File, error) {
	err := os.MkdirAll(kctx.config.TempPath, 0700)
	if err != nil {
		log.Printf("Couldn't create temp folder: %s", err)
		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
		return nil, err
	}
	tempfile, err := os.CreateTemp(kctx.config.TempPath, "kland_upload_")
	if err != nil {
		log.Printf("Couldn't open temp file: %s", err)
		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
		return nil, err
	}
	_, err = io.Copy(tempfile, r)
	if err != nil {
		tempfile.Close()
		log.Printf("Couldn't write temp file: %s", err)
		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
		return nil, err
	}
	_, err = tempfile.Seek(0, io.SeekStart)
	if err != nil {
		tempfile.Close()
		log.Printf("Couldn't seek temp file: %s", err)
		http.Error(w, "Can't write temp file", http.StatusInternalServerError)
		return nil, err
	}
	return tempfile, nil
}

// This function reads the entirety of the data in 'file' stream and puts it in the
// final destination, giving it a random name with the extension appended. The full
// filename is returned (without the path)
func (kctx *KlandContext) RegisterUpload(file io.ReadSeeker, extension string) (string, error) {
	// Before doing anything, check the size of the destination. If it's too big, return an error
	if kctx.config.MaxTotalDataSize > 0 || kctx.config.MaxTotalFileCount > 0 {
		size, count, err := utils.GetTotalDirectorySize(kctx.config.DataPath)
		if err != nil {
			return "", err
		}
		if kctx.config.MaxTotalDataSize > 0 && size >= kctx.config.MaxTotalDataSize {
			return "", &utils.OutOfSpaceError{
				Allowed: kctx.config.MaxTotalDataSize,
				Current: size,
				Units:   "bytes",
			}
		}
		if kctx.config.MaxTotalFileCount > 0 && count >= kctx.config.MaxTotalFileCount {
			return "", &utils.OutOfSpaceError{
				Allowed: kctx.config.MaxTotalFileCount,
				Current: count,
				Units:   "files",
			}
		}
	}
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}
	kctx.pinsmu.Lock()
	defer kctx.pinsmu.Unlock()
	filename, err := kctx.GenerateRandomUniqueFilename(extension)
	dest := filepath.Join(kctx.config.ImagePath(), filename)
	newfile, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(newfile, file)
	if err != nil {
		return "", err
	}
	//log.Printf("Moved %s to %s", path, destabs)
	return filename, nil
}
