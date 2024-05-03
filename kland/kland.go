package kland

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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
	DefaultIpp      = 20
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
			threadViews := kctx.CheckThreadsConvert(threads, err, w)
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
			postViews := kctx.CheckPostsConvert(posts, err, w)
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
				data["publicLink"] = fmt.Sprintf("%s/image?view=%s", kctx.config.RootPath, utils.Unpointer(thread.hash, ""))
				posts, err := GetPaginatedPosts(db, int64(thread.tid), iquery.Page-1, iquery.IPP)
				postViews := kctx.CheckPostsConvert(posts, err, w)
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
			if r.FormValue("url") != "" {
				http.Error(w, "Admin tasks not currently reimplemented", http.StatusTeapot)
				return
			}
			db, err := kctx.config.OpenDb()
			if err != nil {
				reportDbError(err, w)
				return
			}
			defer db.Close()
			//realRedirect := utils.StringToBool(r.FormValue("redirect"))
			//realShort := utils.StringToBool(r.FormValue("shorturl"))
			//imageUrl := ""
			//finalImageName := ""
			ipaddress := r.Header.Get(kctx.config.IpHeader)
			if ipaddress == "" {
				ipaddress = "unknown"
			}
			bucket := r.FormValue("bucket")
			bucketThread, err := kctx.GetOrCreateBucketThread(db, bucket)
			if err != nil {
				log.Printf("Couldn't get bucket thread on upload: %s", err)
				http.Error(w, "Couldn't get bucket thread", http.StatusInternalServerError)
				return
			}
			w.Write([]byte(fmt.Sprintf("%v", bucketThread)))
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
