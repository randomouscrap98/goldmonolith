package webstream

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/randomouscrap98/goldmonolith/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/gorilla/schema"
)

const (
	RoomNameError = "Room name has invalid characters! Try something simpler!"
)

// Query the user sends in to get parts of a stream or whatever
type StreamQuery struct {
	Start       int  `schema:"start"`
	Count       int  `schema:"count"`
	Nonblocking bool `schema:"nonblocking"`
	Readonlykey bool `schema:"readonlykey"`
}

func GetDefaultStreamQuery() *StreamQuery {
	return &StreamQuery{
		Start:       0,
		Count:       -1,
		Nonblocking: false,
		Readonlykey: false,
	}
}

// Result of a stream completion (often times the user only uses
// the data portion)
type StreamResult struct {
	Data        string `json:"data"`
	Readonlykey string `json:"readonlykey"`
	Signalled   int    `json:"signalled"`
	Used        int    `json:"used"`
	Limit       int    `json:"limit"`
}

// The constants you return from the /constants endpoint,
// received from the config (don't want to give out the whole config)
type StreamConstants struct {
	MaxStreamSize  int `json:"maxStreamSize"`
	MaxSingleChunk int `json:"maxSingleChunk"`
}

// All the data held onto for the duration of hosting the webstream
// service (unique instance created for each handler, be careful)
type WebstreamContext struct {
	roomRegex  *regexp.Regexp
	decoder    *schema.Decoder
	obfuscator *utils.ObfuscatedKeys
	backer     *WebStreamBacker_File
	webstreams map[string]*WebStream
	wslock     sync.Mutex
	config     *Config
}

// Produce a new webstream context for hosting webstream
func NewWebstreamContext(config *Config) (*WebstreamContext, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	backer, err := NewFileBacker(config)
	if err != nil {
		return nil, err
	}
	return &WebstreamContext{
		config:     config,
		decoder:    schema.NewDecoder(),
		backer:     backer,
		roomRegex:  roomRegex,
		obfuscator: utils.GetDefaultObfuscation(),
		webstreams: make(map[string]*WebStream),
	}, nil
}

// Get a ready-to-use instance of of a webstream, usable with reads
// and writes immediately, auto-backed by whatever backer is there
func (wc *WebstreamContext) GetStream(name string) *WebStream {
	wc.wslock.Lock()
	defer wc.wslock.Unlock()
	ws, ok := wc.webstreams[name]
	if !ok {
		ws = NewWebStream(name, wc.backer)
		refreshed, err := ws.RefreshStream()
		if err != nil {
			log.Printf("ERROR: Couldn't load webstream %s: %s", name, err)
		}
		if refreshed {
			log.Printf("First load of stream %s from filesystem", name)
		}
		wc.webstreams[name] = ws
	}
	return ws
}

// Taken almost verbatim from the c# program
func (wc *WebstreamContext) GetStreamResult(w http.ResponseWriter, r *http.Request) (*StreamResult, error) {
	room := chi.URLParam(r, "room")
	query := GetDefaultStreamQuery()
	err := wc.decoder.Decode(query, r.URL.Query())
	if err != nil {
		log.Printf("Bad request: %s", err)
		http.Error(w, "Couldn't parse request", http.StatusBadRequest)
		return nil, err
	}
	if query.Readonlykey {
		room, err = wc.obfuscator.GetFromObfuscatedKey(room)
		if err != nil {
			log.Printf("Room not found: %s", err)
			http.Error(w, "Readonly room not found", http.StatusNotFound)
			return nil, err
		}
	}
	if !wc.roomRegex.MatchString(room) {
		http.Error(w, RoomNameError, http.StatusBadRequest)
		return nil, fmt.Errorf(RoomNameError)
	}

	ws := wc.GetStream(room)
	rname := wc.obfuscator.GetObfuscatedKey(room)

	var cancel context.Context = nil
	var cancelfunc context.CancelFunc = func() {}
	if !query.Nonblocking {
		cancel, cancelfunc = context.WithTimeout(r.Context(), time.Duration(wc.config.ReadTimeout))
	}
	defer cancelfunc()

	rawdata, err := ws.ReadData(query.Start, query.Count, cancel)
	if err != nil {
		log.Printf("Error during ReadData: %s", err)
		http.Error(w, "Error while reading data (sorry!)", http.StatusInternalServerError)
		return nil, err
	}

	// Note: that "Signalled" count is very inaccurate, but it was inaccurate on the old
	// c# system so I think it's fine
	return &StreamResult{
		Limit:       wc.config.StreamDataLimit,
		Readonlykey: rname,
		Data:        string(rawdata), // This is expensive I think??
		Signalled:   max(ws.GetListenerCount(), ws.GetLastWriteListenerCount()),
		Used:        ws.GetLength(),
	}, nil
}

func (wc *WebstreamContext) DumpStreams(force bool) {
	wc.wslock.Lock()
	defer wc.wslock.Unlock()
	for k, v := range wc.webstreams {
		if force || time.Now().Sub(v.GetLastWrite()) > time.Duration(wc.config.IdleRoomTime) {
			if force {
				log.Printf("FORCE DUMPING STREAM: %s", k)
			}
			ok, err := v.DumpStream(true)
			if err != nil {
				// A warning is about all we can do...
				log.Printf("WARN: Error saving webstream %s: %s\n", k, err)
			}
			if ok {
				log.Printf("Dumped room %s to filesystem\n", k)
			}
		}
	}
}

func (wc *WebstreamContext) RunBackground(cancel context.Context, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Duration(wc.config.IdleRoomTime))
		defer ticker.Stop()
		log.Printf("Webstream background service started\n")
		for {
			select {
			case <-cancel.Done():
				log.Printf("Webstream background cancelled, exiting + dumping streams\n")
				wc.DumpStreams(true) // SUPER IMPORTANT!
				return
			case <-ticker.C:
				wc.DumpStreams(false)
			}
		}
	}()
}

func (webctx *WebstreamContext) GetHandler() http.Handler {
	r := chi.NewRouter()

	r.Get("/constants", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, StreamConstants{
			MaxStreamSize:  webctx.config.StreamDataLimit,
			MaxSingleChunk: webctx.config.SingleDataLimit,
		})
	})

	r.Get("/{room}", func(w http.ResponseWriter, r *http.Request) {
		result, err := webctx.GetStreamResult(w, r)
		//log.Printf("Result: %p\n", result)
		if err == nil {
			render.PlainText(w, r, result.Data)
		}
	})

	r.Get("/{room}/json", func(w http.ResponseWriter, r *http.Request) {
		result, err := webctx.GetStreamResult(w, r)
		if err == nil {
			render.JSON(w, r, result)
		}
	})

	r.Post("/{room}", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, int64(webctx.config.SingleDataLimit))
		// config.SimpleFormLimit))
		room := chi.URLParam(r, "room")
		if !webctx.roomRegex.MatchString(room) {
			http.Error(w, RoomNameError, http.StatusBadRequest)
			return
		}
		// We're safe to just "read all" since we've limited the body above
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Read POST body error for room %s: %s\n", room, err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		ws := webctx.GetStream(room)
		err = ws.AppendData(data)
		if err != nil {
			log.Printf("Append error for room %s: %s\n", room, err)
			// This COULD be because the room is full, we should show the error
			// (even if it might expose some sensitive info... whatever)
			http.Error(w, fmt.Sprintf("Couldn't append to room: %s", err), http.StatusBadRequest)
			return
		}
	})

	return r
}
