package route

import (
	"4bit.api/v0/server/route/ping"
	"4bit.api/v0/server/route/telegram"
	mux "github.com/gorilla/mux"
)

func InitRootRoute(r *mux.Router) {
	// Ping endpoint.
	pingSubrouter := r.PathPrefix("/ping").Subrouter()
	ping.CreateRoute(pingSubrouter)

	return r
}
