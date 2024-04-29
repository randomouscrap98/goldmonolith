package webstream

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

// A snapshot of information about a webstream. For informational purposes only;
// data is immediately stale as soon as snapshot is made
type WebStreamInfo struct {
	Length                 int
	Capacity               int
	ListenerCount          int
	LastWrite              time.Time
	LastWriteListenerCount int
	Dirty                  bool
}

// Single webstream, tightly coupled with the WebStreamSystem
type webStream struct {
	data               []byte
	mu                 sync.Mutex
	readSignal         chan struct{}
	length             int       // Length of data (even if data has been cleared for mem saving)
	listeners          int       // Amount of listeners currently active
	lastWrite          time.Time // Time of last write to this webstream
	dirty              bool      // There's some change here
	lastWriteListeners int       // Count of listeners at last signal (write)
}

func newWebStream(data []byte) *webStream {
	return &webStream{
		data:       data,
		readSignal: make(chan struct{}),
	}
}

func (ws *webStream) getInfoNoLock() *WebStreamInfo {
	return &WebStreamInfo{
		Length:                 ws.length,
		Capacity:               cap(ws.data),
		ListenerCount:          ws.listeners,
		LastWriteListenerCount: ws.lastWriteListeners,
		LastWrite:              ws.lastWrite,
		Dirty:                  ws.dirty,
	}
}

type WebStreamSystem struct {
	roomRegex   *regexp.Regexp        // Regex to limit room names
	backer      WebStreamBacker       // The backing system to persist streams
	webstreams  map[string]*webStream // ALL streams the system has ever seen at runtime
	wsmu        sync.Mutex            // Lock for webstreams object
	config      *Config
	activeCount int        // Number of active rooms
	acmu        sync.Mutex // lock for activeCount
}

func NewWebStreamSystem(config *Config, backer WebStreamBacker) (*WebStreamSystem, error) {
	roomRegex, err := regexp.Compile(config.RoomRegex)
	if err != nil {
		return nil, err
	}
	// WARN: This may seem ridiculous, but we preload EVERY room from the system. This shouldn't
	// be too bad, since it's just metadata and we're talking rooms in the thousands, not
	// millions (I think...). This simplifies a great number of things, but if this needs
	// to be changed, it should be doable...
	webstreams := make(map[string]*webStream)
	err = backer.BackingIterator(func(k string, gl func() int) bool {
		webstreams[k] = newWebStream(nil)
		webstreams[k].length = gl()
		return true
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

func (wsys *WebStreamSystem) RoomCount() int {
	wsys.acmu.Lock()
	defer wsys.acmu.Unlock()
	return len(wsys.webstreams)
}

func (wsys *WebStreamSystem) atActiveCapacity() bool {
	wsys.acmu.Lock()
	defer wsys.acmu.Unlock()
	return wsys.activeCount >= wsys.config.ActiveRoomLimit
}

func (wsys *WebStreamSystem) incActiveCount() {
	wsys.acmu.Lock()
	wsys.activeCount += 1
	wsys.acmu.Unlock()
}

func (wsys *WebStreamSystem) decActiveCount() {
	wsys.acmu.Lock()
	wsys.activeCount -= 1
	wsys.acmu.Unlock()
}

// Retrieve the ready-made stream object for the given name. Will load
// stream from persistent storage if this is a brand new room, regardless of
// what might be done with it in the future. If stream object already exists,
// you'll get whatever is available (may be a dumped(idle) room...)
func (wsys *WebStreamSystem) getStream(name string) (*webStream, error) {
	if !wsys.roomRegex.MatchString(name) {
		return nil, &RoomNameError{Regex: wsys.roomRegex.String()}
	}
	wsys.wsmu.Lock()
	defer wsys.wsmu.Unlock()
	ws, ok := wsys.webstreams[name]
	if !ok {
		// This is a new room, we must first check if there's enough space to add it...
		if len(wsys.webstreams) >= wsys.config.TotalRoomLimit {
			return nil, &RoomLimitError{Limit: wsys.config.TotalRoomLimit}
		}
		// We're fine to add it, and I don't think there's any need to refresh it
		ws = newWebStream(nil)
		wsys.webstreams[name] = ws
	}
	return ws, nil
}

// Bring the backing back into the stream. It is safe to call this even if the stream
// is already active; it will NOT pull from the backing store again. This does mean
// the backing store can become desynchronized with the in-memory store; this is
// fine for our purposes, as there are many undefines for "changing a backing store
// out from under listeners"
func (wsys *WebStreamSystem) refreshStreamNoLock(name string, ws *webStream) (bool, error) {
	if cap(ws.data) > 0 {
		// Nothing to do, stream has data
		return false, nil
	}
	// Can't refresh if there's too many active rooms
	if wsys.atActiveCapacity() {
		return false, &ActiveRoomLimitError{Limit: wsys.config.ActiveRoomLimit}
	}
	// This ALWAYS loads the stream into memory.
	stream, _, err := wsys.backer.Read(name, wsys.config.StreamDataLimit)
	if err != nil {
		return false, err
	}
	ws.length = len(stream)
	ws.data = stream
	wsys.incActiveCount()
	return true, nil
}

// Dump data from all streams which are idling and still have data. Alternatively, force
// dump every single room with data. Will always clear any dumped stream to conserve memory
func (wsys *WebStreamSystem) DumpStreams(force bool) []string {
	dumped := make([]string, 0)
	wsys.wsmu.Lock()
	defer wsys.wsmu.Unlock()
	idleTime := time.Duration(wsys.config.IdleRoomTime)
	for k, ws := range wsys.webstreams {
		// Lock for the duration of dump checking, you MUST not randomly unlock!!
		ws.mu.Lock()
		if force || time.Now().Sub(ws.lastWrite) > idleTime {
			if force {
				log.Printf("FORCE DUMPING STREAM: %s", k)
			}
			// Only dump if there's really something to dump
			if cap(ws.data) != 0 {
				var err error
				var written bool
				// Only write if the data is dirty (to save disk writes? idk...)
				if ws.dirty {
					err = wsys.backer.Write(k, ws.data)
					if err != nil {
						// A warning is about all we can do...
						log.Printf("WARN: Error saving webstream %s: %s\n", k, err)
					} else {
						written = true
					}
				}
				// Only CLEAR the data if nothing bad happened
				if err == nil {
					ws.data = nil
					ws.dirty = false
					wsys.decActiveCount()
					dumped = append(dumped, k)
					if written {
						log.Printf("Dumped room %s to persistent storage\n", k)
					} else {
						log.Printf("Offloaded room %s (no change)\n", k)
					}
				}
			}
		}
		ws.mu.Unlock()
	}
	return dumped
}

// Append the given data to this stream. Will throw an error if the
// stream overflows the capacity
func (wsys *WebStreamSystem) AppendData(name string, data []byte) error {
	ws, err := wsys.getStream(name)
	if err != nil {
		return err
	}
	// Lock for the ENTIRE duration of the append, including refresh. The
	// system doesn't work if you refresh then randomly lose it!
	ws.mu.Lock()
	defer ws.mu.Unlock()
	// Data MUST be available, do a refresh
	refreshed, err := wsys.refreshStreamNoLock(name, ws)
	if err != nil {
		return err
	}
	if refreshed {
		log.Printf("Write for %s at %d+%d refreshed backing stream\n", name, ws.length, len(data))
	}
	if len(data)+ws.length > cap(ws.data) {
		return &OverCapacityError{Capacity: cap(ws.data)}
	}
	ws.data = ws.data[:ws.length+len(data)] // Embiggen
	copy(ws.data[ws.length:], data)         // we don't use append because we specifically do not want it to grow ever
	// Keep track of all the little data
	ws.length = len(ws.data)
	ws.lastWrite = time.Now()
	ws.lastWriteListeners = ws.listeners
	ws.dirty = true
	// Signal to all the readers that something is ready
	close(ws.readSignal)
	ws.readSignal = make(chan struct{})
	return nil
}

// This function will safely read from the given webstream, blocking if
// you're trying to read past the end of the data. You can cancel it with the
// given context. If the context is nil, the function is NONBLOCKING
func (wsys *WebStreamSystem) ReadData(name string, start, length int, cancel context.Context) ([]byte, error) {
	if start < 0 {
		// This is what the other service did, mmm want to make it as similar as possible
		return nil, fmt.Errorf("start must be non-zero")
	}
	ws, err := wsys.getStream(name)
	if err != nil {
		return nil, err
	}
	ws.mu.Lock()
	// This should "just work" to give a relatively accurate listener count
	ws.listeners += 1
	defer func() { ws.listeners -= 1 }()
	// In this special situation, we must simply wait until the data becomes available.
	// It is also OK if the data is not currently backed, since we're just waiting on
	// a signal and not actually reading anything.
	if start >= ws.length {
		if cancel != nil {
			// We're still locked at this point, so we know nobody is changing this out
			// from under us
			waiter := ws.readSignal
			ws.mu.Unlock()
			select {
			case <-waiter:
				// We were signalled, recursively call us again for easiness
				return wsys.ReadData(name, start, length, cancel)
			case <-cancel.Done():
				// We were killed, but we DON'T throw the error? Is that OK??
				return nil, nil
			}
			// We should always exit this if statement with a return...
		} else {
			ws.mu.Unlock()
			// This is the "nonblocking" part of reading at the end of the stream
			return nil, nil
		}
	}
	// If we get here, we know that we have data to read. Data can only ever grow
	// (also we're in a lock so we know the length is static at this point).
	defer ws.mu.Unlock()
	// Also, since we're ACTUALLY reading, we must have the data available, so refresh
	refreshed, err := wsys.refreshStreamNoLock(name, ws)
	if err != nil {
		return nil, err
	}
	if refreshed {
		log.Printf("Read for %s at %d+%d refreshed backing stream\n", name, start, length)
	}
	// The previous service changed the length to fit within the bounds, so a read
	// near the end with some ridiculous length would only returrn up to the end of the stream.
	// We replicate that here with the same exact data massaging
	if length < 0 || length > ws.length-start {
		length = ws.length - start
	}
	// I don't really care if people up top mess around with the data,
	// just return a simple slice
	return ws.data[start : start+length], nil
}

func (wsys *WebStreamSystem) RoomInfo(name string) (*WebStreamInfo, error) {
	ws, err := wsys.getStream(name)
	if err != nil {
		return nil, err
	}
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.getInfoNoLock(), nil
}
