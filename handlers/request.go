package handlers

import (
	"fmt"
	"net/http"
	"regexp"
)

// jsonpCallbackPattern matches valid JavaScript identifiers, including
// dotted and bracketed property access (e.g. "foo.bar[0]")
var jsonpCallbackPattern = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*(\.[A-Za-z_$][A-Za-z0-9_$]*|\[[0-9]+\])*$`)

// scheme returns the underlying URL scheme of the original request.
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	if scheme := r.Header.Get("X-Forwarded-Protocol"); scheme != "" {
		return scheme
	}
	if ssl := r.Header.Get("X-Forwarded-Ssl"); ssl == "on" {
		return "https"
	}
	if scheme := r.Header.Get("X-Url-Scheme"); scheme != "" {
		return scheme
	}
	return "http"
}

type handlerFunc func(http.ResponseWriter, *http.Request) (int, error)

// wrapJSONP writes b (JSON marshalled to bytes) as a JSONP response to
// w if the callback query parameter is present, and writes b as a JSON
// response otherwise. Any error that occurs during writing is returned.
func wrapJSONP(w http.ResponseWriter, r *http.Request, b []byte) (err error) {
	callback := r.URL.Query().Get("callback")

	if callback != "" {
		if !jsonpCallbackPattern.MatchString(callback) {
			http.Error(w, "Invalid callback parameter", http.StatusBadRequest)
			return fmt.Errorf("invalid JSONP callback parameter")
		}
		w.Header().Set("Content-Type", "application/javascript")
		_, err = w.Write([]byte(fmt.Sprintf("%s(%s);", callback, b)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	return
}
