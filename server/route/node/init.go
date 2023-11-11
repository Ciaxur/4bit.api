package node

import (
	"context"

	"github.com/gorilla/mux"
)

// Creates all routes for the Node endpoint.
func CreateRoutes(ctx *context.Context, r *mux.Router) {
	CreateNodeRoute(r)
	CreateStateRoute(r)
}
