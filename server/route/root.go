package route

import (
	"4bit.api/v0/server/route/ping"
	mux "github.com/gorilla/mux"
)

func InitRootRoute() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)

	pingSubrouter := r.PathPrefix("/ping").Subrouter()
	ping.InitPingRoute(pingSubrouter)

	return r
}
