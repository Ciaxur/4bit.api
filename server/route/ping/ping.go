package ping

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func CreateRoute(ctx *context.Context, r *mux.Router) {
	r.HandleFunc("", pingHandler).Methods("GET")
}
