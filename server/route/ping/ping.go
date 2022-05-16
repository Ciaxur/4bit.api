package ping

import (
	"net/http"

	"github.com/gorilla/mux"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func CreateRoute(r *mux.Router) {
	r.HandleFunc("", pingHandler).Methods("GET")
}
