package webstream

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Streams are in-memory for maximum performance and minimum complexity.
// However, they can periodically be dumped to a "backer" for
// permanent (or otherwise) storage
type WebStreamBacker interface {
	Write(string, []byte) error
	Read(string) ([]byte, error)
	Exists(string) bool
}

// A webstream is a chunk of preallocated memory that can be read from and appended to.
// This webstream understands that it is backed by a file, and that it is possible to
// remove the memory while still functioning
type WebStream struct {
	stream     []byte
	mu         sync.Mutex
	readSignal chan struct{}
	length     int
	listeners  int
	lastWrite  time.Time
	Name       string
	Backer     WebStreamBacker
}

func NewWebStream(name string, backer WebStreamBacker) *WebStream {
	return &WebStream{
		Name:       name,
		Backer:     backer,
		readSignal: make(chan struct{}),
	}
}

func (ws *WebStream) GetListenerCount() int {
	// Do we REALLY need to lock on this? IDK...
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.listeners
}

func (ws *WebStream) GetLastWrite() time.Time {
	// Do we REALLY need to lock on this? IDK...
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.lastWrite
}

func (ws *WebStream) GetLength() int {
	// Do we REALLY need to lock on this? IDK...
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.length
}

// Append the given data to this stream. Will throw an error if the
// stream overflows the capacity
func (ws *WebStream) AppendData(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	// Data MUST be available, do a refresh
	refreshed, err := ws.refreshStream()
	if err != nil {
		return err
	}
	if refreshed {
		log.Printf("Write for %s at %d+%d refreshed backing stream\n", ws.Name, ws.length, len(data))
	}
	if len(data)+ws.length > cap(ws.stream) {
		return fmt.Errorf("data overflows capacity: %d", cap(ws.stream))
	}
	ws.stream = ws.stream[:ws.length+len(data)] // Embiggen
	copy(ws.stream[ws.length:], data)           // we don't use append because we specifically do not want it to grow ever
	ws.length = len(ws.stream)
	ws.lastWrite = time.Now()
	close(ws.readSignal)
	ws.readSignal = make(chan struct{})
	return nil
}

// This function will safely read from the given webstream, blocking if
// you're trying to read past the end of the data. You can cancel it with the
// given context. If the context is nil, the function is NONBLOCKING
func (ws *WebStream) ReadData(start, length int, cancel context.Context) ([]byte, error) {
	if start < 0 {
		// This is what the other service did, mmm want to make it as similar as possible
		return nil, fmt.Errorf("start must be non-zero")
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
			// But now the waiter could be in any state, which... should be fine?
			select {
			case <-waiter:
				// We were signalled
				return ws.ReadData(start, length, cancel)
			case <-cancel.Done():
				// We were killed
				return nil, cancel.Err()
			}
			// We should always exit this if statement with a return...
		} else {
			// This is the "nonblocking" part of reading at the end of the stream
			return nil, nil
		}
	}
	// If we get here, we know that we have data to read. Data can only ever grow
	// (also we're in a lock so we know the length is static at this point).
	defer ws.mu.Unlock()
	// Also, since we're ACTUALLY reading, we must have the data available, so refresh
	refreshed, err := ws.refreshStream()
	if err != nil {
		return nil, err
	}
	if refreshed {
		log.Printf("Read for %s at %d+%d refreshed backing stream\n", ws.Name, start, length)
	}
	// The previous service changed the length to fit within the bounds, so a read
	// near the end with some ridiculous length would only returrn up to the end of the stream.
	// We replicate that here with the same exact data massaging
	if length < 0 || length > ws.length-start {
		length = ws.length - start
	}
	// I don't really care if people up top mess around with the data,
	// just return a simple slice
	return ws.stream[start : start+length], nil
}

// Bring the backing back into the stream. It is safe to call this even if the stream
// is already active; it will NOT pull from the backing store again. This does mean
// the backing store can become desynchronized with the in-memory store; this is
// fine for our purposes, as there are many undefines for "changing a backing store
// out from under listeners"
func (ws *WebStream) refreshStream() (bool, error) {
	if cap(ws.stream) > 0 {
		return false, nil
	}
	stream, err := ws.Backer.Read(ws.Name)
	if err != nil {
		return false, err
	}
	ws.length = len(stream)
	ws.stream = stream
	return true, nil
}

// Public interface for refreshStream (with locking)
func (ws *WebStream) RefreshStream() (bool, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.refreshStream()
}

// Dump the stream back to the backing file, and optionally remove
// the memory (length will still be available)
func (ws *WebStream) DumpStream(clear bool) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if cap(ws.stream) == 0 {
		return fmt.Errorf("can't dump stream: nothing in memory")
	}
	err := ws.Backer.Write(ws.Name, ws.stream)
	if err != nil {
		return err
	}
	// At this point, we know it's all good
	if clear {
		ws.stream = nil
	}
	return nil
}
