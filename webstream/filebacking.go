package webstream

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Backing data storage for WebStream that stores to the filesystem based
// on the configured location
type WebStreamBacker_File struct {
	// This is a GLOBAL mutex: I'm EXTREMELY limiting the filesystem operations
	// such that only one can happen at a time (on purpose)
	mu     sync.Mutex
	Config *Config
}

func (wb *WebStreamBacker_File) fpath(name string) string {
	return filepath.Join(wb.Config.StreamFolder, name)
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

func (wb *WebStreamBacker_File) Read(name string) ([]byte, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	//gen := func() []byte { return make([]byte, 0, wb.Config.StreamDataLimit) } // Create the WHOLE THING. This may change...
	backing, err := os.Open(wb.fpath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// This is normal: we don't want a non-existent file to throw an error,
			// just let the caller think they have something...
			return make([]byte, 0, wb.Config.StreamDataLimit), nil //gen(), nil
		}
		return nil, err
	}
	defer backing.Close()
	length, err := backing.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	stream := make([]byte, length, wb.Config.StreamDataLimit) //gen()
	_, err = io.ReadFull(backing, stream)                     //[:length])
	if err != nil {
		return nil, err
	}
	return stream, nil
}
