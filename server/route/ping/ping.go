package ping

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received %s request for host %s from IP address %s and X-FORWARDED-FOR %s",
		r.Method, r.Host, r.RemoteAddr, r.Header.Get("X-FORWARDED-FOR"))
	w.Write([]byte("pong"))
}

func InitPingRoute(r *mux.Router) {
	r.StrictSlash(true)
	r.HandleFunc("/", pingHandler).Methods("GET")
}
