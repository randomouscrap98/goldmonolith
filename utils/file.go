package utils

import (
	"io"
	"net/http"
)

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
