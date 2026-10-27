package dtos

type HealthStatus string

const HealthStatusOK HealthStatus = "ok"

type HealthResponse struct {
	Status HealthStatus `json:"status" enum:"ok"`
}

type HealthOutput struct {
	Body HealthResponse
}
