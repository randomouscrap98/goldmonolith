package kland

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/randomouscrap98/goldmonolith/utils"
)

const (
	Version       = "0.1.0"
	AdminIdKey    = "adminId"
	IsAdminKey    = "isAdmin"
	PostStyleKey  = "postStyle"
	ImageEndpoint = "/img"
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
						MaxAge: int(time.Duration(kctx.config.CookieExpire).Seconds()),
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
			r.ParseMultipartForm(kctx.config.MaxMultipartMemory)
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
					realraw, _, err := ParseImageDataUrl(form.raw)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					outfile, err = kctx.WriteTemp(realraw, w)
					if err != nil {
						return
					}
					log.Printf("Read image from base64 string")
					// Don't need to close outfile, we're going to use it later, THEY can close it...
				} else if form.animation != "" {
					tempfile, err := kctx.MakeTemp(w)
					if err != nil {
						return
					}
					err = ConvertAnimation(form.animation, tempfile)
					if err != nil {
						tempfile.Close()
						http.Error(w, fmt.Sprintf("Couldn't decode json: %s", err), http.StatusBadRequest)
						return
					}
					outfile = tempfile
				}
			}
			defer CloseDeleteUploadFile(outfile)
			ctype, err := utils.DetectContentType(outfile)
			if strings.Index(ctype, "image") != 0 {
				http.Error(w, "Server rejected file: couldn't detect image format!", http.StatusBadRequest)
				return
			}
			extension, err := utils.FirstErr(mime.ExtensionsByType(ctype))
			if err != nil {
				http.Error(w, fmt.Sprintf("Server rejected file: %s", err), http.StatusBadRequest)
				return
			}
			// Now we can generate a random name and move the file
			finalname, err := kctx.RegisterUpload(outfile, *extension)
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
				imageUrl = fmt.Sprintf("%s/%s", kctx.config.ShortUrl, finalname)
			} else {
				imageUrl = fmt.Sprintf("%s%s%s/%s", kctx.config.FullUrl, kctx.config.RootPath, ImageEndpoint, finalname)
			}

			log.Printf("Image url: %s", imageUrl)

			if form.redirect {
				http.Redirect(w, r, imageUrl, http.StatusSeeOther)
			} else {
				w.Write([]byte(imageUrl))
			}
		})
	})

	// --- Static files -----
	var err error
	err = utils.FileServer(r, ImageEndpoint+"/", kctx.config.ImagePath(), false)
	if err != nil {
		return nil, err
	}
	err = utils.FileServer(r, "/anm", kctx.config.TextPath(), false)
	if err != nil {
		return nil, err
	}
	err = utils.FileServer(r, "/", kctx.config.StaticFilePath, true)
	if err != nil {
		return nil, err
	}
	return r, nil
}
