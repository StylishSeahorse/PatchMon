package commands

import (
	"errors"
	"testing"
)

type fakeReboot struct {
	needed    bool
	execErr   error
	checks    int
	reboots   int
	lastDelay int
}

func (f *fakeReboot) CheckRebootRequired() (bool, string) {
	f.checks++
	return f.needed, "kernel updated"
}

func (f *fakeReboot) ExecuteReboot(delay int, _ string) error {
	f.reboots++
	f.lastDelay = delay
	return f.execErr
}

func TestMaybeRebootAfterPatch(t *testing.T) {
	boom := errors.New("patch failed")
	cases := []struct {
		name       string
		fake       fakeReboot
		requested  bool
		dryRun     bool
		runErr     error
		want       bool
		wantChecks int
	}{
		{"not requested", fakeReboot{needed: true}, false, false, nil, false, 0},
		{"dry run", fakeReboot{needed: true}, true, true, nil, false, 0},
		{"failed run", fakeReboot{needed: true}, true, false, boom, false, 0},
		{"host does not need a reboot", fakeReboot{needed: false}, true, false, nil, false, 1},
		{"needs a reboot", fakeReboot{needed: true}, true, false, nil, true, 1},
		{"reboot cannot be scheduled", fakeReboot{needed: true, execErr: boom}, true, false, nil, false, 1},
	}
	for _, c := range cases {
		f := c.fake
		if got := maybeRebootAfterPatch(&f, c.requested, c.dryRun, c.runErr); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
		if f.checks != c.wantChecks {
			t.Errorf("%s: reboot checks = %d, want %d", c.name, f.checks, c.wantChecks)
		}
		wantReboots := 0
		if c.want || c.fake.execErr != nil {
			wantReboots = 1
		}
		if f.reboots != wantReboots {
			t.Errorf("%s: reboot calls = %d, want %d", c.name, f.reboots, wantReboots)
		}
		if f.reboots > 0 && f.lastDelay != postPatchRebootDelayMinutes {
			t.Errorf("%s: delay = %d, want %d", c.name, f.lastDelay, postPatchRebootDelayMinutes)
		}
	}
}
