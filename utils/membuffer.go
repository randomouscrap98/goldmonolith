package utils

// taken from https://stackoverflow.com/a/73679110/1066474

import (
	"errors"
	"fmt"
	"io"
)

// Implements io.ReadWriteSeeker for testing purposes.
type MemBuffer struct {
	buffer []byte
	offset int64
}

// Creates new buffer that implements io.ReadWriteSeeker for testing purposes.
func NewMemBuffer(initial []byte) MemBuffer {
	if initial == nil {
		initial = make([]byte, 0, 100)
	}
	return MemBuffer{
		buffer: initial,
		offset: 0,
	}
}

func (fb *MemBuffer) Bytes() []byte {
	return fb.buffer
}

func (fb *MemBuffer) Len() int {
	return len(fb.buffer)
}

func (fb *MemBuffer) Read(b []byte) (int, error) {
	available := len(fb.buffer) - int(fb.offset)
	if available == 0 {
		return 0, io.EOF
	}
	size := len(b)
	if size > available {
		size = available
	}
	copy(b, fb.buffer[fb.offset:fb.offset+int64(size)])
	fb.offset += int64(size)
	return size, nil
}

func (fb *MemBuffer) Write(b []byte) (int, error) {
	copied := copy(fb.buffer[fb.offset:], b)
	if copied < len(b) {
		fb.buffer = append(fb.buffer, b[copied:]...)
	}
	fb.offset += int64(len(b))
	return len(b), nil
}

func (fb *MemBuffer) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = fb.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(fb.buffer)) + offset
	default:
		return 0, errors.New("Unknown Seek Method")
	}
	if newOffset > int64(len(fb.buffer)) || newOffset < 0 {
		return 0, fmt.Errorf("Invalid Offset %d", offset)
	}
	fb.offset = newOffset
	return newOffset, nil
}
