package webstream

import (
	"fmt"
	"sync"
	"time"
)

type webStream struct {
	data               []byte
	mu                 sync.Mutex
	readSignal         chan struct{}
	length             int       // Meta length of data (even if data has been cleared for mem saving)
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

// Simple dump function which will dump the data in the given stream to the given
// function in a threadsafe manner, optionally clearing the data after
func (ws *webStream) dumpStream(writer func([]byte) error, clear bool) (bool, error) {
	// Don't let other stuff touch the current stream
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if cap(ws.data) == 0 {
		return false, nil
	}
	err := writer(ws.data) //wsys.backer.Write(name, stream.Data)
	if err != nil {
		return false, err
	}
	ws.dirty = false
	// At this point, we know it's all good
	if clear {
		ws.data = nil
	}
	return true, nil
}

// Simple append function which adds the data to the given webstream.
// It is always assumed we have our data in-memory (let someone else manage that)
func (ws *webStream) appendData(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if len(data)+ws.length > cap(ws.data) {
		return fmt.Errorf("data overflows capacity: %d", cap(ws.data))
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
