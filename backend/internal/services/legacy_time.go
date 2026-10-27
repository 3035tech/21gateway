package services

import (
	"time"
	_ "time/tzdata"
)

const legacyQueryDateLayout = "2006-01-02 15:04:05-07"
const legacyQueryTimeout = 5 * time.Second

var legacyQueryLocation = func() *time.Location {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(err)
	}
	return location
}()

// Separate calendar boundaries preserve historical days of 23 or 25 hours.
func legacyDayRange(timestamp int64) (time.Time, time.Time) {
	local := time.Unix(timestamp, 0).In(legacyQueryLocation)
	y, m, d := local.Date()
	start := legacyDayStart(y, m, d)
	end := legacyDayStart(y, m, d+1)
	return start.UTC(), end.UTC()
}
func legacyDayStart(year int, month time.Month, day int) time.Time {
	// Noon normalizes month/year rollover without entering a midnight transition.
	noon := time.Date(year, month, day, 12, 0, 0, 0, legacyQueryLocation)
	y, m, d := noon.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, legacyQueryLocation)
	if start.Day() != d {
		start = start.Add(time.Hour)
	} // Sao Paulo's skipped midnight.
	return start
}
