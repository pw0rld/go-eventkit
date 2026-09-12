package dateparser

import (
	"testing"
	"time"
)

func TestInvalidTimesAndOverflow(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	for _, s := range []string{"next monday 25:00", "sep 15 nonsense", "3:99pm", "next friday at bad", "in 999999999999999999999 hours", "999999999999999999999 days ago"} {
		if got, err := ParseDateRelativeTo(s, now); err == nil {
			t.Errorf("%q accepted as %s", s, got)
		}
	}
	for _, s := range []string{"999999999999999999999h", "2562048h", "106752d"} {
		if got, err := ParseAlertDuration(s); err == nil {
			t.Errorf("%q accepted as %s", s, got)
		}
	}
	if _, err := ParseDateRelativeTo("today", now, WithDefaultHour(24)); err == nil {
		t.Fatal("invalid default hour")
	}
}
