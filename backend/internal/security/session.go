package security

import "time"

const RegularSessionIdleTimeout = 8 * time.Hour
const RememberedSessionLifetime = 14 * 24 * time.Hour
const RememberedSessionIdleTimeout = RememberedSessionLifetime
