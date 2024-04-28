package webstream

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
	//"github.com/randomouscrap98/goldmonolith/utils"
)

type webStream struct {
	Data               []byte
	Mu                 sync.Mutex
	ReadSignal         chan struct{}
	Length             int       // Meta length of data (even if data has been cleared for mem saving)
	Listeners          int       // Amount of listeners currently active
	LastWrite          time.Time // Time of last write to this webstream
	LastWriteListeners int       // Count of listeners at last signal (write)
}

// Simple dump function which will dump the data in the given stream to the given
// function in a threadsafe manner, optionally clearing the data after
func (ws *webStream) dumpStream(writer func([]byte) error, clear bool) (bool, error) {
	// Don't let other stuff touch the current stream
	ws.Mu.Lock()
	defer ws.Mu.Unlock()
	if cap(ws.Data) == 0 {
		return false, nil
	}
	err := writer(ws.Data) //wsys.backer.Write(name, stream.Data)
	if err != nil {
		return false, err
	}
	// At this point, we know it's all good
	if clear {
		ws.Data = nil
	}
	return true, nil
}

type WebStreamSystem struct {
	mu          sync.Mutex            // General lock for whole webstream system
	roomRegex   *regexp.Regexp        // Regex to limit room names
	backer      WebStreamBacker       // The backing system to persist streams
	webstreams  map[string]*webStream // ALL streams the system has ever seen at runtime
	activeCount int                   // Amount of "active" streams (streams with data)
	config      *Config
	//IdleTimeout time.Duration         // How long before a stream dumps its memory (not its metadata)
}

func NewWebStreamSet(config *Config, backer WebStreamBacker) (*WebStreamSystem, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	return &WebStreamSystem{
		roomRegex:  roomRegex,
		backer:     backer,
		webstreams: make(map[string]*webStream),
		config:     config,
		//IdleTimeout: time.Duration(config.IdleRoomTime),
	}, nil
}

// Retrieve the ready-made stream object for the given name. Will load
// stream from persistent storage if this is a brand new room, regardless of
// what might be done with it in the future. If stream object already exists,
// you'll get whatever is available (may be a dumped(idle) room...)
func (wsys *WebStreamSystem) getStream(name string) (*webStream, error) {
	if !wsys.roomRegex.MatchString(name) {
		return nil, fmt.Errorf("Room name has invalid characters! Try something simpler!")
	}
	wsys.mu.Lock()
	defer wsys.mu.Unlock()
	ws, ok := wsys.webstreams[name]
	if !ok {
		// Getting the stream data is ALWAYS a refresh, so check active count
		if wsys.activeCount >= wsys.config.ActiveRoomLimit {
			return nil, fmt.Errorf("active room limit reached (%d), must wait for another room to idle", wsys.config.ActiveRoomLimit)
		}
		// Go get the stream data first.
		wsdata, exists, err := wsys.backer.Read(name, wsys.config.StreamDataLimit)
		if err != nil {
			return nil, err
		}
		// Have to do something serious if this is a new room. Don't want users to
		// create too many rooms on the filesystem
		if !exists {
			dcount, err := wsys.backer.Count()
			if err != nil {
				return nil, err
			}
			if dcount >= wsys.config.TotalRoomLimit {
				return nil, fmt.Errorf("room limit reached (%d), no new rooms can be created", wsys.config.TotalRoomLimit)
			}
		}
		ws = &webStream{
			Data:       wsdata,
			ReadSignal: make(chan struct{}),
		}
		wsys.webstreams[name] = ws
		wsys.activeCount += 1
	}
	return ws, nil
}

// Dump data from all streams which are idling and still have data. Alternatively, force
// dump every single room with data. Will always clear any dumped stream to conserve memory
func (wsys *WebStreamSystem) DumpStreams(force bool) {
	wsys.mu.Lock()
	defer wsys.mu.Unlock()
	for k, v := range wsys.webstreams {
		if force || time.Now().Sub(v.LastWrite) > time.Duration(wsys.config.IdleRoomTime) {
			if force {
				log.Printf("FORCE DUMPING STREAM: %s", k)
			}
			ok, err := v.dumpStream(func(d []byte) error {
				return wsys.backer.Write(k, d)
			}, true)
			if err != nil {
				// A warning is about all we can do...
				log.Printf("WARN: Error saving webstream %s: %s\n", k, err)
			}
			if ok {
				wsys.activeCount -= 1
				log.Printf("Dumped room %s to filesystem (%d still active)\n", k, wsys.activeCount)
			}
		}
	}
}
