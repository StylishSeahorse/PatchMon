package store

import (
	"testing"
	"time"
)

func TestValidateRecurrenceCron(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expr string
		ok   bool
	}{
		{"0 1 * * *", true},
		{"30 23 * * 0", true},
		{"15 3 * * 1-5", true},
		{"0 2 1 * *", true},
		{"  0   1 * * * ", true},
		{"* * * * *", false},
		{"*/5 * * * *", false},
		{"0 * * * *", false},
		{"0 */2 * * *", false},
		{"0,30 1 * * *", false},
		{"60 1 * * *", false},
		{"0 24 * * *", false},
		{"0 1 * *", false},
		{"0 1 * * * *", false},
		{"@daily", false},
		{"@every 1h", false},
		{"0 1 32 * *", false},
		{"", false},
	}
	for _, c := range cases {
		err := ValidateRecurrenceCron(c.expr)
		if (err == nil) != c.ok {
			t.Errorf("ValidateRecurrenceCron(%q) err=%v, want ok=%v", c.expr, err, c.ok)
		}
	}
}

func TestNextRecurrenceReadsScheduleInOrgTimezone(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	got, err := NextRecurrence("0 1 * * *", "Europe/London", from)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.January, 11, 1, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("London winter: got %v, want %v", got, want)
	}

	got, err = NextRecurrence("0 1 * * *", "America/New_York", from)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.January, 11, 6, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("New York winter: got %v, want %v", got, want)
	}
}

func TestNextRecurrenceKeepsWallClockAcrossDST(t *testing.T) {
	t.Parallel()
	// UK clocks go forward on 2026-03-29, so 03:30 local moves from 03:30Z to 02:30Z.
	start := time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)
	got, err := NextRecurrence("30 3 * * *", "Europe/London", start)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.March, 28, 3, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("before change: got %v, want %v", got, want)
	}
	got, err = NextRecurrence("30 3 * * *", "Europe/London", got)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.March, 29, 2, 30, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("after change: got %v, want %v", got, want)
	}
}

// A time that does not exist on the spring-forward day (01:00-02:00 in London)
// is skipped for that day rather than shifted; the docs say so.
func TestNextRecurrenceSkipsNonexistentLocalTime(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)
	got, err := NextRecurrence("0 1 * * *", "Europe/London", from)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNextRecurrenceWeekday(t *testing.T) {
	t.Parallel()
	// 2026-01-10 is a Saturday; the next Sunday 03:00 UTC is 2026-01-11.
	from := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	got, err := NextRecurrence("0 3 * * 0", "UTC", from)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.January, 11, 3, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNextRecurrenceUnknownZoneFallsBackToUTC(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	got, err := NextRecurrence("0 1 * * *", "Not/AZone", from)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, time.January, 11, 1, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
