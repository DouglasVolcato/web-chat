package helpers

import "net/http"

func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	url = PathURL(url)

	if IsHTMXRequest(r) {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}
