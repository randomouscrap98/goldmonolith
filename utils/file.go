package utils

import (
	"io"
	"net/http"
	"os"
)

// Wrapper arond http.DetectContentType, since it's nontrivial to get the first
// 512 bytes in a reader (unfortunately)
func DetectContentType(reader io.ReadSeeker) (string, error) {
	dctbuf := make([]byte, 512)
	_, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}
	_, err = io.ReadFull(reader, dctbuf)
	if err != nil {
		return "", err
	}
	return http.DetectContentType(dctbuf), nil
}

// Check to see if any of the given paths exist. If any throw an error
// that isn't "not found", that error is rethrown
func CheckAnyPathExists(paths []string) (bool, error) {
	for _, p := range paths {
		_, err := os.Stat(p)
		if err == nil { // File was found
			return true, nil
		} else if !os.IsNotExist(err) { // File was ERROR
			return false, err
		}
	}
	return false, nil
}
