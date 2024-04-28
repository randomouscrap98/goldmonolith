package webstream

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
	//"github.com/randomouscrap98/goldmonolith/utils"
)

type WebStreamSystem struct {
	mu         sync.Mutex            // General lock for whole webstream system
	roomRegex  *regexp.Regexp        // Regex to limit room names
	backer     WebStreamBacker       // The backing system to persist streams
	webstreams map[string]*webStream // ALL streams the system has ever seen at runtime
	//activeCount int                   // Amount of "active" streams (streams with data)
	config *Config
}

func NewWebStreamSet(config *Config, backer WebStreamBacker) (*WebStreamSystem, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	// WARN: This may seem ridiculous, but we preload EVERY room from the system. This shouldn't
	// be too bad, since it's just metadata and we're talking rooms in the thousands, not
	// millions (I think...). This simplifies a great number of things, but if this needs
	// to be changed, it should be doable...
	webstreams := make(map[string]*webStream)
	err = backer.BackingIterator(func(k string) bool {
		webstreams[k] = newWebStream(nil)
		return false
	})
	if err != nil {
		return nil, err
	}
	return &WebStreamSystem{
		roomRegex:  roomRegex,
		backer:     backer,
		webstreams: webstreams,
		config:     config,
	}, nil
}

// A threadsafe check if we're at or exceeding active room capacity
func (wsys *WebStreamSystem) atActiveCapacity() bool {
	wsys.mu.Lock()
	defer wsys.mu.Unlock()
	count := 0
	for _, ws := range wsys.webstreams {
		if cap(ws.data) > 0 {
			count += 1
			if count >= wsys.config.ActiveRoomLimit {
				return true
			}
		}
	}
	return false
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
		// This is a new room, we must first check if there's enough space to add it...
		if len(wsys.webstreams) >= wsys.config.TotalRoomLimit {
			return nil, fmt.Errorf("room limit reached (%d), no new rooms can be created", wsys.config.TotalRoomLimit)
		}
		// We're fine to add it, and I don't think there's any need to refresh it
		ws = newWebStream(nil)
		wsys.webstreams[name] = ws

		// Getting the stream data is ALWAYS a refresh, so check active count
		// if wsys.activeCount >= wsys.config.ActiveRoomLimit {
		// 	return nil, fmt.Errorf("active room limit reached (%d), must wait for another room to idle", wsys.config.ActiveRoomLimit)
		// }
		// // Go get the stream data first.
		// wsdata, exists, err := wsys.backer.Read(name, wsys.config.StreamDataLimit)
		// if err != nil {
		// 	return nil, err
		// }
		// Have to do something serious if this is a new room. Don't want users to
		// create too many rooms on the filesystem
		// if !exists {
		// 	dcount, err := wsys.backer.Count()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	if dcount >= wsys.config.TotalRoomLimit {
		// 		return nil, fmt.Errorf("room limit reached (%d), no new rooms can be created", wsys.config.TotalRoomLimit)
		// 	}
		// }
		// ws = &webStream{
		// 	data:       wsdata,
		// 	readSignal: make(chan struct{}),
		// }
		// wsys.webstreams[name] = ws
		// wsys.activeCount += 1
	}
	return ws, nil
}

// Bring the backing back into the stream. It is safe to call this even if the stream
// is already active; it will NOT pull from the backing store again. This does mean
// the backing store can become desynchronized with the in-memory store; this is
// fine for our purposes, as there are many undefines for "changing a backing store
// out from under listeners"
func (wsys *WebStreamSystem) refreshStream(name string, ws *webStream) (bool, error) {
	if cap(ws.data) > 0 {
		// Nothing to do, stream has data
		return false, nil
	}
	//wsys.mu.Lock()
	//defer wsys.mu.Unlock()
	//activeCount := wsys.activeCount()
	// Can't refresh if there's too many active rooms
	if wsys.atActiveCapacity() { //wsys.activeCount >= wsys.config.ActiveRoomLimit {
		return false, fmt.Errorf("active room limit reached (%d), must wait for another room to idle", wsys.config.ActiveRoomLimit)
	}
	// This ALWAYS loads the stream into memory.
	stream, _, err := wsys.backer.Read(name, wsys.config.StreamDataLimit)
	if err != nil {
		return false, err
	}
	ws.length = len(stream)
	ws.data = stream
	return true, nil
}

// Dump data from all streams which are idling and still have data. Alternatively, force
// dump every single room with data. Will always clear any dumped stream to conserve memory
func (wsys *WebStreamSystem) DumpStreams(force bool) {
	wsys.mu.Lock()
	defer wsys.mu.Unlock()
	for k, v := range wsys.webstreams {
		if force || time.Now().Sub(v.lastWrite) > time.Duration(wsys.config.IdleRoomTime) {
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
				//wsys.activeCount -= 1
				log.Printf("Dumped room %s to filesystem\n", k) //, wsys.activeCount)
			}
		}
	}
}

// Append the given data to this stream. Will throw an error if the
// stream overflows the capacity
func (wsys *WebStreamSystem) AppendData(name string, data []byte) error {
	ws, err := wsys.getStream(name)
	if err != nil {
		return err
	}
	// Data MUST be available, do a refresh
	refreshed, err := wsys.refreshStream(name, ws)
	if err != nil {
		return err
	}
	if refreshed {
		log.Printf("Write for %s at %d+%d refreshed backing stream\n", name, ws.length, len(data))
	}
	return ws.appendData(data)
}
