package webstream

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
	//"github.com/randomouscrap98/goldmonolith/utils"
)

// Streams are in-memory for maximum performance and minimum complexity.
// However, they can periodically be dumped to a "backer" for
// permanent (or otherwise) storage
type WebStreamBacker interface {
	Write(string, []byte) error
	Read(string) ([]byte, error)
	Exists(string) bool
}

type webStream struct {
	Stream             []byte
	Mu                 sync.Mutex
	ReadSignal         chan struct{}
	Length             int
	Listeners          int
	LastWrite          time.Time
	LastWriteListeners int // Count of listeners at last signal (write)
	//Name               string
	//Backer             WebStreamBacker
}

type WebStreamSet struct {
	roomRegex   *regexp.Regexp
	backer      WebStreamBacker
	webstreams  map[string]*webStream
	mu          sync.Mutex
	IdleTimeout time.Duration
	//config     *Config
}

func NewWebStreamSet(config *Config, backer WebStreamBacker) (*WebStreamSet, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	return &WebStreamSet{
		//config:     config,
		backer:      backer,
		roomRegex:   roomRegex,
		webstreams:  make(map[string]*webStream),
		IdleTimeout: time.Duration(config.IdleRoomTime),
	}, nil
}

// Dump the stream back to the backing file, and optionally remove
// the memory (length will still be available)
func (ws *WebStreamSet) dumpStream(clear bool) (bool, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if cap(ws.stream) == 0 {
		return false, nil //fmt.Errorf("can't dump stream: nothing in memory")
	}
	err := ws.Backer.Write(ws.Name, ws.stream)
	if err != nil {
		return false, err
	}
	// At this point, we know it's all good
	if clear {
		ws.stream = nil
	}
	return true, nil
}

// Get a ready-to-use instance of of a webstream, usable with reads
// and writes immediately, auto-backed by whatever backer is there
func (ss *StreamSet) getStream(name string) (*WebStream, error) {
	if !ss.roomRegex.MatchString(name) {
		return nil, fmt.Errorf("Room name has invalid characters! Try something simpler!")
	}
	ss.wslock.Lock()
	defer ss.wslock.Unlock()
	ws, ok := ss.webstreams[name]
	if !ok {
		ws = NewWebStream(name, ss.backer)
		refreshed, err := ws.RefreshStream()
		if err != nil {
			log.Printf("ERROR: Couldn't load webstream %s: %s", name, err)
		}
		if refreshed {
			log.Printf("First load of stream %s from persistent backing", name)
		}
		ss.webstreams[name] = ws
	}
	return ws, nil
}

func (ss *StreamSet) DumpStreams(force bool) {
	ss.wslock.Lock()
	defer ss.wslock.Unlock()
	for k, v := range ss.webstreams {
		if force || time.Now().Sub(v.GetLastWrite()) > ss.IdleTimeout {
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
