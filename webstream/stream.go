package webstream

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// A webstream is a chunk of preallocated memory that can be read from and appended to.
// This webstream understands that it is backed by a file, and that it is possible to
// remove the memory while still functioning
type WebStream struct {
	stream      []byte
	mu          sync.Mutex
	length      int64
	Capacity    int64
	BackingFile string
}

// This function will safely read from the given webstream, blocking if
// you're trying to read past the end of the data. You can cancel it with the
// given context (required)
func (ws *WebStream) ReadData(start, length int64, cancel context.Context) ([]byte, error) {
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
		log.Printf("Read for %d+%d refreshed backing stream\n", start, length)
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
// is already active; it will NOT pull from the backing store again
func (ws *WebStream) refreshStream() (bool, error) {
	if cap(ws.stream) > 0 {
		return false, nil
	}
	backing, err := os.Open(ws.BackingFile) //ws.StreamBacker()
	if err != nil {
		return false, err
	}
	defer backing.Close()
	ws.length, err = backing.Seek(0, io.SeekEnd)
	ws.stream = make([]byte, ws.length, ws.Capacity)
	_, err = io.ReadFull(backing, ws.stream)
	if err != nil {
		return false, err
	}
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
	writer, err := os.Create(ws.BackingFile)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = writer.Write(ws.stream)
	if err != nil {
		return err
	}
	// At this point, we know it's all good
	if clear {
		ws.stream = nil
	}
	return nil
}

//func
