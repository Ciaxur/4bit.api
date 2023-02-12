package node

import "github.com/gorilla/mux"

// Creates all routes for the Node endpoint.
func CreateRoutes(r *mux.Router) {
	CreateNodeRoute(r)
	CreateStateRoute(r)
}
