package services

import (
	"testing"
	"time"
)

func TestCallbackBackoffBounded(t *testing.T) {
	var previous time.Duration
	for i := 1; i <= 100; i++ {
		value := CallbackBackoff(i)
		if value < initialCallbackBackoff || value > maxCallbackBackoff+maxCallbackBackoff/4 {
			t.Fatal("unbounded callback delay", value)
		}
		if i < 8 && previous > value {
			t.Fatal("backoff regressed")
		}
		previous = value
	}
}
