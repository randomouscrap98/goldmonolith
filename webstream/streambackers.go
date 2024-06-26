package webstream

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Streams are in-memory for maximum performance and minimum complexity.
// However, they can periodically be dumped to a "backer" for
// permanent (or otherwise) storage
type WebStreamBacker interface {
	// Write the given data to the backing at the given string. Does a full overwrite
	Write(string, []byte) error
	// Returns the full backing data, letting you know if it previous existed or not.
	// Pass the capacity for the newly created byte array
	Read(string, int) ([]byte, bool, error)
	// Repeatedly calls your given function for each backing available in the system.
	// Useful for searches or otherwise
	BackingIterator(func(string, func() int) bool) error
}

func Exists(b WebStreamBacker, name string) (bool, error) {
	exists := false
	err := b.BackingIterator(func(k string, gl func() int) bool {
		if k == name {
			exists = true
			return false // stop
		}
		return true // Continue
	})
	return exists, err
}

// --- FILE: Simple file based backer ---

// Backing data storage for WebStream that stores to the filesystem based
// on the configured location
type WebStreamBacker_File struct {
	// This is a GLOBAL mutex: I'm EXTREMELY limiting the filesystem operations
	// such that only one can happen at a time (on purpose)
	mu     sync.Mutex
	Folder string
}

func NewFileBacker(folder string) (*WebStreamBacker_File, error) {
	err := os.MkdirAll(folder, 0750)
	if err != nil {
		return nil, err
	}
	return &WebStreamBacker_File{
		Folder: folder,
	}, nil
}

func (wb *WebStreamBacker_File) fpath(name string) string {
	return filepath.Join(wb.Folder, name)
}

func (wb *WebStreamBacker_File) Write(name string, data []byte) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	writer, err := os.Create(wb.fpath(name))
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = writer.Write(data)
	return err
}

func (wb *WebStreamBacker_File) Read(name string, capacity int) ([]byte, bool, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	backing, err := os.Open(wb.fpath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// File doesn't exist, create memory and tell the caller there's no file
			return make([]byte, 0, capacity), false, nil
		}
		return nil, false, err
	}
	defer backing.Close()
	stat, err := backing.Stat()
	if err != nil {
		return nil, false, err
	}
	stream := make([]byte, stat.Size(), capacity)
	_, err = io.ReadFull(backing, stream)
	if err != nil {
		return nil, false, err
	}
	return stream, true, nil
}

func (wb *WebStreamBacker_File) BackingIterator(callback func(string, func() int) bool) error {
	d, err := os.ReadDir(wb.Folder)
	if err != nil {
		return err
	}
	for _, de := range d {
		getLength := func() int {
			info, err := de.Info()
			if err == nil {
				return int(info.Size())
			} else {
				return 0
			}
		}
		if !callback(de.Name(), getLength) {
			return nil
		}
	}
	return nil
}

// --- MEM: A backer for testing, in-memory storage only ---

type backerEvent struct {
	Type int    // Read 0 write 1
	Data []byte // This is wasteful but like whatever
}

type testBacker struct {
	Rooms  map[string][]byte
	Events []backerEvent
}

func NewTestBacker() *testBacker {
	return &testBacker{
		Rooms:  make(map[string][]byte),
		Events: make([]backerEvent, 0),
	}
}

func (tb *testBacker) Write(name string, data []byte) error {
	tb.Rooms[name] = data
	tb.Events = append(tb.Events, backerEvent{
		Type: 1,
		Data: data,
	})
	return nil
}

func (tb *testBacker) Read(name string, capacity int) ([]byte, bool, error) {
	data, ok := tb.Rooms[name]
	tb.Events = append(tb.Events, backerEvent{
		Type: 0,
		Data: data,
	})
	if !ok { // This is a "new" room, so give it something...
		return make([]byte, 0, capacity), false, nil
	} else {
		return data, true, nil
	}
}

func (tb *testBacker) BackingIterator(callback func(string, func() int) bool) error {
	for k, v := range tb.Rooms {
		if !callback(k, func() int { return len(v) }) {
			return nil
		}
	}
	return nil
}
