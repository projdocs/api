package tus

import (
	"net/http"
	"strings"
)

// responseInterceptor wraps gin's ResponseWriter to capture the
// Upload-Location header that tusd sets on a successful POST.
type responseInterceptor struct {
	http.ResponseWriter
	uploadID string
}

func (r *responseInterceptor) WriteHeader(status int) {
	if status == http.StatusCreated {
		loc := r.ResponseWriter.Header().Get("Location")
		// Location is the full URL; the upload ID is the last path segment.
		if loc != "" {
			parts := strings.Split(strings.TrimRight(loc, "/"), "/")
			r.uploadID = parts[len(parts)-1]
		}
	}
	r.ResponseWriter.WriteHeader(status)
}
