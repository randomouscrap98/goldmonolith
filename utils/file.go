package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Wrapper arond http.DetectContentType, since it's nontrivial to get the first
// 512 bytes in a reader (unfortunately)
func DetectContentType(reader io.ReadSeeker) (string, error) {
	dctbuf := make([]byte, 512)
	_, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}
	length, err := io.ReadFull(reader, dctbuf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", err
	}
	dctbuf = dctbuf[:length]
	_, err = reader.Seek(0, io.SeekStart)
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

// Return the total size of the directory, walking through all folders
// recursively. Also returns a total file count. IDK how performant this is...
func GetTotalDirectorySize(path string) (int64, int64, error) {
	var size int64
	var count int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
			count += 1
		}
		return err
	})
	return size, count, err
}
