package queue

import (
	"testing"
	"time"
)

func TestPatchPolicyOccurrenceMissed(t *testing.T) {
	t.Parallel()
	due := time.Date(2026, time.January, 11, 1, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"on time", due, false},
		{"next tick", due.Add(time.Minute), false},
		{"edge of window", due.Add(patchPolicyCatchUpWindow), false},
		{"just past window", due.Add(patchPolicyCatchUpWindow + time.Second), true},
		{"server was down all night", due.Add(9 * time.Hour), true},
	}
	for _, c := range cases {
		if got := patchPolicyOccurrenceMissed(c.now, due); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
