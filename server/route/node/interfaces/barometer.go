package interfaces

// Power consumption data.
type PowerState struct {
	Current_mA  float32
	LoadVoltage float32
	Power_mW    float32
}

// Barometer data.
type BarometerState struct {
	Pressure    float32
	Temperature float32
	Altitude    float32
}
