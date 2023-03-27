package route

import (
	"4bit.api/v0/server/route/camera"
	"4bit.api/v0/server/route/node"
	"4bit.api/v0/server/route/ping"
	"4bit.api/v0/server/route/telegram"
	mux "github.com/gorilla/mux"
)

func InitRootRoute(r *mux.Router) error {
	// Ping endpoint.
	pingSubrouter := r.PathPrefix("/ping").Subrouter()
	ping.CreateRoute(pingSubrouter)

	// Telegram endpoint.
	telegramMessageSubrouter := r.PathPrefix("/telegram").Subrouter()
	telegram.CreateRoute(telegramMessageSubrouter)

	// Node endpoint.
	nodeSubrouter := r.PathPrefix("/node").Subrouter()
	node.CreateRoutes(nodeSubrouter)

	// Camera endpoint.
	cameraSubrouter := r.PathPrefix("/camera").Subrouter()
	if err := camera.CreateRoutes(cameraSubrouter); err != nil {
		return err
	}

	return nil
}
