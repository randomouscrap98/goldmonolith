package kland

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version            = "0.1.0"
	AdminIdKey         = "adminId"
	IsAdminKey         = "isAdmin"
	PostStyleKey       = "postStyle"
	LongCookie         = 365 * 24 * 60 * 60
	DefaultIpp         = 20
	MaxMultipartMemory = 256_000
)

func reportDbError(err error, w http.ResponseWriter) {
	log.Printf("ERROR OPENING DB: %s", err)
	http.Error(w, "Error opening database", http.StatusInternalServerError)
}

func checkSingleThread(threads []Thread, err error, w http.ResponseWriter) bool {
	if err != nil {
		log.Printf("ERROR RETRIEVING THREADS: %s", err)
		http.Error(w, "Error retrieving thread", http.StatusInternalServerError)
		return false
	}
	if len(threads) < 1 {
		http.Error(w, "Thread not found", http.StatusNotFound)
		return false
	}
	return true
}

// Query the user can send to image index
type GetImageQuery struct {
	Bucket string `schema:"bucket"`
	AsJSON bool   `schema:"asJSON"`
	Page   int    `schema:"page"`
	IPP    int    `schema:"ipp"`
	View   string `schema:"view"`
}

// Store query into the given data. It's unfortunately nontrivial...
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

type UploadImageQuery struct {
	raw       string
	animation string
	redirect  bool
	short     bool
	ipaddress string
	bucket    string
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
			threadViews := kctx.ConvertThreadResult(threads, err, w)
			if threadViews == nil {
				return
			}
			data := kctx.GetDefaultData(r)
			data["threads"] = threadViews
			kctx.RunTemplate("index.tmpl", w, data)
		})

		r.Get("/thread/{id}", func(w http.ResponseWriter, r *http.Request) {
			db, err := kctx.config.OpenDb()
			if err != nil {
				reportDbError(err, w)
				return
			}
			defer db.Close()
			idraw := chi.URLParam(r, "id")
			tid, err := strconv.ParseInt(idraw, 10, 64)
			if err != nil {
				http.Error(w, "Bad file ID format", http.StatusBadRequest)
				return
			}
			threads, err := GetThreadsById(db, []int64{tid})
			if !checkSingleThread(threads, err, w) {
				return
			}
			posts, err := GetPostsInThread(db, tid)
			postViews := kctx.ConvertPostResult(posts, err, w)
			if postViews == nil {
				return
			}
			data := kctx.GetDefaultData(r)
			data["thread"] = ConvertThread(threads[0], kctx.config)
			data["posts"] = postViews
			kctx.RunTemplate("thread.tmpl", w, data)
		})

		r.Get("/image", func(w http.ResponseWriter, r *http.Request) {
			db, err := kctx.config.OpenDb()
			if err != nil {
				reportDbError(err, w)
				return
			}
			defer db.Close()

			iquery, err := kctx.ParseImageQuery(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("Query parse error: %s", err), http.StatusBadRequest)
				return
			}

			data := kctx.GetDefaultData(r)
			iquery.IntoData(data)
			// Note: we used to have "hideuploads", we don't use that anymore, but just in case...
			data["hideuploads"] = false

			var thread *Thread
			getByField := func(name string, value string) bool {
				threads, err := GetThreadsByField(db, name, value)
				if err != nil {
					log.Printf("Couldn't get thread: %s", err)
					http.Error(w, "Couldn't get thread?", http.StatusInternalServerError)
					return false
				}
				if len(threads) >= 1 {
					thread = &threads[0]
				}
				return true
			}

			if iquery.View != "" {
				if !getByField("hash", iquery.View) {
					return
				}
				data["readonly"] = true
			} else {
				if !getByField("subject", bucketSubject(iquery.Bucket)) {
					return
				}
			}

			if thread != nil {
				data["publicLink"] = fmt.Sprintf("%s/image?view=%s", kctx.config.RootPath, thread.hash)
				posts, err := GetPaginatedPosts(db, int64(thread.tid), iquery.Page-1, iquery.IPP)
				postViews := kctx.ConvertPostResult(posts, err, w)
				if postViews == nil {
					return
				}
				data["pastImages"] = postViews
			} else {
				data["isnewthread"] = true
			}

			if iquery.AsJSON {
				utils.RespondJson(data, w, nil)
			} else {
				kctx.RunTemplate("image.tmpl", w, data)
			}
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
			adminid := r.FormValue("adminid")
			poststyle := r.FormValue("poststyle")
			redirect := r.FormValue("redirect")
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
			log.Printf("Redirect: %s", redirect)
			http.Redirect(w, r, redirect, http.StatusSeeOther)
		})

		r.Post("/uploadimage", func(w http.ResponseWriter, r *http.Request) {
			// WE want to parse the form so we can set the mem size...
			r.ParseMultipartForm(MaxMultipartMemory)
			defer func() {
				err := r.MultipartForm.RemoveAll()
				if err != nil {
					log.Printf("ERROR REMOVING TEMP FORM FILE: %s", err)
				}
			}()
			// Set limits on the body
			r.Body = http.MaxBytesReader(w, r.Body, int64(kctx.config.MaxImageSize))
			if r.FormValue("url") != "" {
				http.Error(w, "Admin tasks not currently reimplemented (url upload)", http.StatusTeapot)
				return
			}
			db, err := kctx.config.OpenDb()
			if err != nil {
				reportDbError(err, w)
				return
			}
			defer db.Close()
			form := kctx.ParseImageUploadQuery(r)
			bucketThread, err := kctx.GetOrCreateBucketThread(db, form.bucket)
			if err != nil {
				log.Printf("Couldn't get bucket thread on upload: %s", err)
				http.Error(w, "Couldn't get bucket thread", http.StatusInternalServerError)
				return
			}
			var outfile io.ReadSeekCloser
			outfile, _, err = r.FormFile("image")
			if err != nil { // Couldn't load the form file, it needs to be one of two other things
				if form.raw != "" {
					// This is a base64 thing, pretty simple to parse.
					firstComma := strings.IndexRune(form.raw, ',')
					if firstComma < 0 {
						http.Error(w, "Malformed raw image string (missing comma)", http.StatusBadRequest)
						return
					}
					reader := strings.NewReader(form.raw[firstComma+1:])
					decoder := base64.NewDecoder(base64.StdEncoding, reader)
					outfile, err = kctx.WriteTemp(decoder, w)
					if err != nil {
						return
					}
				} else if form.animation != "" {
					http.Error(w, "Animation decoding not yet supported", http.StatusTeapot)
					return
				}
				// outfile, ok = file.(*os.File)
				// if !ok { // Somehow the file is in memory?
				// 	// The actual file is here, let's put it somewhere
				// 	outfile, err = kctx.WriteTemp(file, w)
				// 	// Because we're here, we can specifically close the file. The file
				// 	// is some in-memory buffer AND we're done with it; we already wrote it to fs
				// 	file.Close()
				// 	if err != nil {
				// 		return
				// 	}
				// 	log.Printf("Uploaded file was in memory; wrote to fs")
				// } else {
				// 	log.Printf("Uploaded file already temp")
				// }
			} //else
			ctype, err := utils.DetectContentType(outfile)
			err = outfile.Close()
			if err != nil {
				log.Printf("CAN'T CLOSE NEW FILE: %s", err)
				http.Error(w, "Couldn't write temp file", http.StatusInternalServerError)
				return
			}
			if strings.Index(ctype, "image") != 0 {
				http.Error(w, "Server rejected file: couldn't detect image format!", http.StatusBadRequest)
				return
			}
			extension, err := utils.FirstErr(mime.ExtensionsByType(ctype))
			if err != nil {
				http.Error(w, fmt.Sprintf("Server rejected file: %s", err), http.StatusBadRequest)
				return
			}
			// Now we can generate a random name and move hte file
			finalname, err := kctx.MoveAndRegisterUpload(outfile.Name(), *extension)
			if err != nil {
				log.Printf("Can't move upload: %s", err)
				http.Error(w, "Couldn't write file", http.StatusInternalServerError)
				return
			}
			_, err = InsertImagePost(db, form.ipaddress, finalname, bucketThread.tid)
			if err != nil {
				log.Printf("CAN'T INSERT POST: %s", err)
				http.Error(w, "Couldn't write post", http.StatusInternalServerError)
				return
			}

			imageUrl := ""
			if form.short {
				imageUrl = fmt.Sprintf("%s/finalname", kctx.config.ShortUrl)
			} else {
				imageUrl = fmt.Sprintf("%s%s/i/%s", kctx.config.FullUrl, kctx.config.RootPath, finalname)
			}

			log.Printf("Image url: %s", imageUrl)

			if form.redirect {
				http.Redirect(w, r, imageUrl, http.StatusSeeOther)
			} else {
				w.Write([]byte(imageUrl)) //fmt.Sprintf("%v", bucketThread)))
			}
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
