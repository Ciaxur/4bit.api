package interfaces

type StateType uint8

const (
	UNKNOWN   StateType = 0
	BAROMETER StateType = 1 // Barometer entiry of a node.
	POWER     StateType = 2 // Energy consumption of a node.
)

// Query the node state.
type StateGetRequest struct {
	Limit *uint64 // Limit the number of entries to query.
	Type  StateType
}

// Create a new node state.
type StatePostRequest struct {
	BarometerState *BarometerState
	Power          *PowerState
}
