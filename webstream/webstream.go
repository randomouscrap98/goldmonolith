package webstream

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"sync"

	"github.com/randomouscrap98/goldmonolith/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/gorilla/schema"
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
	Data        string `schema:"data"`
	Readonlykey string `schema:"readonlykey"`
	Signalled   int    `schema:"signalled"`
	Used        int    `schema:"used"`
	Limit       int    `schema:"limit"`
}

// The constants you return from the /constants endpoint,
// received from the config (don't want to give out the whole config)
type StreamConstants struct {
	MaxStreamSize  int `json:"maxStreamSize"`
	MaxSingleChunk int `json:"maxSingleChunk"`
}

// All the data held onto for the duration of hosting the handler
type WebstreamContext struct {
	roomRegex  *regexp.Regexp
	decoder    *schema.Decoder
	obfuscator *utils.ObfuscatedKeys
	backer     *WebStreamBacker_File
	webstreams map[string]*WebStream
	wslock     sync.Mutex
	//webstreams *WebstreamCollection
	config *Config
}

// Produce a new webstream context for hosting webstream
func NewWebstreamContext(config *Config) (*WebstreamContext, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	backer := GetDefaultFileBacker(config)
	return &WebstreamContext{
		config:     config,
		decoder:    schema.NewDecoder(),
		backer:     backer,
		roomRegex:  roomRegex,
		obfuscator: utils.GetDefaultObfuscation(),
		webstreams: make(map[string]*WebStream), //NewWebstreamCollection(backer),
	}, nil
}

func (wc *WebstreamContext) GetStream(name string) *WebStream {
	wc.wslock.Lock()
	defer wc.wslock.Unlock()
	ws, ok := wc.webstreams[name]
	if !ok {
		ws = NewWebStream(name, wc.backer)
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
		http.Error(w, "Room name has invalid characters! Try something simpler!", http.StatusBadRequest)
		return nil, err
	}

	ws := wc.GetStream(room)
	rname := wc.obfuscator.GetObfuscatedKey(room)

	result := StreamResult{
		Limit:       wc.config.StreamDataLimit,
		Used:        ws.GetLength(),
		Readonlykey: rname,
		Signalled:   0,
	}

	var cancel context.Context = nil
	if !query.Nonblocking {
		cancel = r.Context()
	}

	rawdata, err := ws.ReadData(query.Start, query.Count, cancel)
	if err != nil {
		log.Printf("Error during ReadData: %s", err)
		http.Error(w, "Error while reading data (sorry!)", http.StatusInternalServerError)
		return nil, err
	}

	result.Data = string(rawdata) // This is expensive I think??
	result.Signalled = ws.GetListenerCount()

	return &result, nil
}

func GetHandler(config *Config) http.Handler {
	webctx, err := NewWebstreamContext(config)
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Get("/constants", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, StreamConstants{
			MaxStreamSize:  config.StreamDataLimit,
			MaxSingleChunk: config.SingleDataLimit,
		})
	})

	r.Get("{room}", func(w http.ResponseWriter, r *http.Request) {
		result, err := webctx.GetStreamResult(w, r)
		if err == nil {
			render.JSON(w, r, result.Data)
		}
	})

	r.Get("{room}/json", func(w http.ResponseWriter, r *http.Request) {
		result, err := webctx.GetStreamResult(w, r)
		if err == nil {
			render.JSON(w, r, result)
		}
	})

	return r
}
