package market

import (
	"testing"
	"time"
)

func TestReadyForFullTime(t *testing.T) {
	kick := time.Date(2026, 8, 21, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	if ReadyForFullTime(kick, kick.Add(10*time.Minute)) {
		t.Fatal("too early")
	}
	if !ReadyForFullTime(kick, kick.Add(100*time.Minute)) {
		t.Fatal("should accept after 100m")
	}
}
