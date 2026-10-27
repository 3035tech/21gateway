package security

import "time"

const RegularSessionIdleTimeout = 30 * time.Minute
const RememberedSessionLifetime = 14 * 24 * time.Hour
const RememberedSessionIdleTimeout = 7 * 24 * time.Hour
