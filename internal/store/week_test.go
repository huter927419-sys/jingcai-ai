package store

import (
	"testing"
	"time"
)

func TestWeekStartThursday(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	fri := time.Date(2026, 8, 21, 12, 0, 0, 0, loc)
	got := WeekStart(fri)
	want := time.Date(2026, 8, 20, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	thu := time.Date(2026, 8, 20, 9, 0, 0, 0, loc)
	if !WeekStart(thu).Equal(want) {
		t.Fatalf("thursday %v", WeekStart(thu))
	}
}
