package webstream

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// )

// Simple dump function which will dump the data in the given stream to the given
// function in a threadsafe manner, optionally clearing the data after
// func (ws *webStream) dumpStream(writer func([]byte) error, clear bool) (bool, error) {
// 	// Don't let other stuff touch the current stream
// 	ws.mu.Lock()
// 	defer ws.mu.Unlock()
// }

// Simple append function which adds the data to the given webstream.
// It is always assumed we have our data in-memory (let someone else manage that)
// func (ws *webStream) appendData(data []byte) error {
// 	ws.mu.Lock()
// 	defer ws.mu.Unlock()
// 	if len(data)+ws.length > cap(ws.data) {
// 		return fmt.Errorf("data overflows capacity: %d", cap(ws.data))
// 	}
// 	ws.data = ws.data[:ws.length+len(data)] // Embiggen
// 	copy(ws.data[ws.length:], data)         // we don't use append because we specifically do not want it to grow ever
// 	// Keep track of all the little data
// 	ws.length = len(ws.data)
// 	ws.lastWrite = time.Now()
// 	ws.lastWriteListeners = ws.listeners
// 	ws.dirty = true
// 	// Signal to all the readers that something is ready
// 	close(ws.readSignal)
// 	ws.readSignal = make(chan struct{})
// 	return nil
// }
