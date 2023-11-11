package route

import (
	"context"

	"4bit.api/v0/server/route/camera"
	"4bit.api/v0/server/route/node"
	"4bit.api/v0/server/route/ping"
	"4bit.api/v0/server/route/telegram"
	mux "github.com/gorilla/mux"
)

func InitRootRoute(ctx *context.Context, r *mux.Router) error {
	// Ping endpoint.
	pingSubrouter := r.PathPrefix("/ping").Subrouter()
	ping.CreateRoute(ctx, pingSubrouter)

	// Telegram endpoint.
	telegramMessageSubrouter := r.PathPrefix("/telegram").Subrouter()
	telegram.CreateRoute(ctx, telegramMessageSubrouter)

	// Node endpoint.
	nodeSubrouter := r.PathPrefix("/node").Subrouter()
	node.CreateRoutes(ctx, nodeSubrouter)

	// Camera endpoint.
	cameraSubrouter := r.PathPrefix("/camera").Subrouter()
	if err := camera.CreateRoutes(ctx, cameraSubrouter); err != nil {
		return err
	}

	return nil
}
