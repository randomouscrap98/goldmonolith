package utils

import (
	"encoding/json"
	"net/http"
	"time"
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

func DeleteCookie(name string, w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    name,
		Value:   "",
		Expires: time.Now().Add(-time.Hour),
	})
}

// Return the value from an integer cookie
func GetCookieOrDefault[T any](name string, r *http.Request, def T, parse func(string) (T, error)) T {
	cookie, err := r.Cookie(name)
	if err != nil {
		return def
	}
	parsed, err := parse(cookie.Value)
	//strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		return def
	}
	return parsed
}
