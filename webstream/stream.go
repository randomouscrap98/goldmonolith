package webstream

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Streams are in-memory for maximum performance and minimum complexity.
// However, they can periodically be dumped to a "backer" for
// permanent (or otherwise) storage
type WebStreamBacker interface {
	Write(string, []byte) error
	Read(string) ([]byte, error)
}

// A webstream is a chunk of preallocated memory that can be read from and appended to.
// This webstream understands that it is backed by a file, and that it is possible to
// remove the memory while still functioning
type WebStream struct {
	stream     []byte
	mu         sync.Mutex
	readSignal chan int
	length     int
	Name       string
	Backer     WebStreamBacker
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
	copy(ws.stream[ws.length:], data)
	ws.length += len(data)
	return nil
}

// This function will safely read from the given webstream, blocking if
// you're trying to read past the end of the data. You can cancel it with the
// given context (required)
func (ws *WebStream) ReadData(start, length int, cancel context.Context) ([]byte, error) {
	if start < 0 {
		// This is what the other service did, mmm want to make it as similar as possible
		return nil, fmt.Errorf("start must be non-zero")
	}
	ws.mu.Lock()
	// In this special situation, we must simply wait until the data becomes available.
	// It is also OK if the data is not currently backed, since we're just waiting on
	// a signal and not actually reading anything.
	if start >= ws.length {
		ws.mu.Unlock()
		//TODO: what the hell is this supposed to do
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
