package config

import "time"

const (
	MaximumMORetentionDays     = 36500
	StartupTimeout             = 5 * time.Second
	ShutdownTimeout            = 10 * time.Second
	ReadinessTimeout           = 2 * time.Second
	ReadHeaderTimeout          = 5 * time.Second
	ReadTimeout                = 15 * time.Second
	WriteTimeout               = 30 * time.Second
	IdleTimeout                = 60 * time.Second
	MaxHeaderBytes             = 64 << 10
	MaxBodyBytes               = 1 << 20
	CORSMaxAgeSeconds          = 300
	DatabaseMaxOpenConnections = 20
	DatabaseMaxIdleConnections = 10
	DatabaseConnectionLifetime = 30 * time.Minute
	DatabaseConnectionIdleTime = 5 * time.Minute
	ExitFailure                = 1
)
