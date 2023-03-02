package parking

import (
	"fmt"

	"4bit.api/v0/database"
)

// Retrieves the last known barometric entry for a given node id.
func GetLastKnownBarometerEntry(nodeId uint64) (*database.NodeBarometerState, error) {
	db := database.DbInstance
	barometerStates := []database.NodeBarometerState{}
	if err := db.Model(&barometerStates).
		Relation("Node").
		Where("node_id = ?", nodeId).
		Limit(1).
		Order("timestamp DESC").
		Select(); err != nil {
		return nil, fmt.Errorf("failed to find barometer stat for node %d: %v", nodeId, err)
	}

	if len(barometerStates) == 0 {
		return nil, fmt.Errorf("no entires found for node %d", nodeId)
	}

	return &barometerStates[0], nil
}

func GetParkingFloor(altitude float32) (uint8, error) {
	// Parking Floor 1.
	// non-charging floor 1 -> ~3-4.5m

	// Parking Floor 2.
	// Non-charging Floor 2 -> ~39m.
	if altitude <= 40.0 && altitude > 38.0 {
		return 2, nil
	}

	// Parking Floor 3.
	// NOTE: -30/-35 might be wrong...
	// Charging on Floor 3 -> ~72-74m
	// Non-charging Floor 3 -> ~20m
	if altitude <= 21.0 && altitude > 19.0 {
		return 3, nil
	}
	// Parking Floor 4.
	// Parking Floor 5.

	// Unknown floor.
	return 0, fmt.Errorf("altitude value of %.2f does not have a floor mapped", altitude)
}
