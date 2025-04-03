package respond

import "net/http"

func WithStatusCode(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}
