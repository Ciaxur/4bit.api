package interfaces

import "4bit.api/v0/database"

type ListCamerasRequest struct {
	Limit uint64 `json:"limit"`
}

type ListCameraResponse struct {
	Cameras []database.CameraEntry
}

type AddCameraRequest struct {
	Camera database.CameraEntry
}

type RemoveCameraRequest struct {
	Camera database.CameraEntry
}

type SnapCameraRequest struct {
	IP string `json:"ip"`
}

type CameraResponseBase struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

type SnapCameraResponse struct {
	// Key value pair of each camera ip and it's corresponding buffer.
	Cameras map[string]CameraResponseBase `json:"cameras"`
}

type StreamCameraResponse struct {
	// Key value pair of each camera ip and it's corresponding buffer.
	Cameras map[string]CameraResponseBase `json:"cameras"`
}
