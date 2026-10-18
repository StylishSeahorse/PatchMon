package handler

import "testing"

func TestUpdateAutoReboot(t *testing.T) {
	t.Parallel()
	yes, no := true, false
	cron := "0 3 * * 1"
	cases := []struct {
		name              string
		recurrencePresent bool
		recurrence        *string
		requested         *bool
		wantValue         bool
		wantChange        bool
	}{
		{"nothing sent", false, nil, nil, false, false},
		{"enable alone", false, nil, &yes, true, true},
		{"disable alone", false, nil, &no, false, true},
		{"schedule set and enabled", true, &cron, &yes, true, true},
		{"schedule set, flag left alone", true, &cron, nil, false, false},
		{"clearing the schedule clears it", true, nil, nil, false, true},
		{"clearing the schedule beats an enable request", true, nil, &yes, false, true},
	}
	for _, c := range cases {
		value, change := updateAutoReboot(c.recurrencePresent, c.recurrence, c.requested)
		if value != c.wantValue || change != c.wantChange {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", c.name, value, change, c.wantValue, c.wantChange)
		}
	}
}
