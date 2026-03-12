package helpers

import "net/http"

// IsHTMXRequest returns true when the request was triggered by htmx.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
