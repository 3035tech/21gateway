package dtos

type HealthStatus string

const HealthStatusOK HealthStatus = "ok"
const HealthStatusDegraded HealthStatus = "degraded"

type HealthResponse struct {
	Status HealthStatus `json:"status" enum:"ok"`
}

type HealthOutput struct {
	Body HealthResponse
}

type DependencyStatus string

const (
	DependencyOK          DependencyStatus = "ok"
	DependencyUnavailable DependencyStatus = "unavailable"
	DependencyDisabled    DependencyStatus = "disabled"
)

type DependenciesResponse struct {
	Status   HealthStatus     `json:"status" enum:"ok,degraded"`
	Database DependencyStatus `json:"database" enum:"ok,unavailable"`
	SMTP     DependencyStatus `json:"smtp" enum:"ok,unavailable,disabled"`
	SQS      DependencyStatus `json:"sqs" enum:"ok,unavailable,disabled"`
}

type DependenciesOutput struct {
	Status int
	Body   DependenciesResponse
}
