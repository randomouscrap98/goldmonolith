package utils

import (
	"encoding/json"
	"net/http"
)

func FinalError(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RespondPlaintext(data []byte, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write(data)
	FinalError(err, w)
}

func RespondJson(v any, w http.ResponseWriter, extra func(*json.Encoder)) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	if extra != nil {
		extra(encoder)
	}
	err := encoder.Encode(v)
	FinalError(err, w)
}
