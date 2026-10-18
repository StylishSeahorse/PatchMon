package system

import (
	"reflect"
	"strings"
	"testing"
)

func TestRebootCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		goos     string
		delay    int
		reason   string
		wantName string
		wantArgs []string
	}{
		{"linux one minute", "linux", 1, "policy", "shutdown", []string{"-r", "+1", "policy"}},
		{"linux now", "linux", 0, "policy", "shutdown", []string{"-r", "now", "policy"}},
		{"freebsd uses the same form", "freebsd", 5, "policy", "shutdown", []string{"-r", "+5", "policy"}},
		{"delay is clamped high", "linux", 600, "x", "shutdown", []string{"-r", "+60", "x"}},
		{"negative delay becomes now", "linux", -3, "x", "shutdown", []string{"-r", "now", "x"}},
		{"blank reason gets a default", "linux", 1, "  ", "shutdown", []string{"-r", "+1", "Triggered by PatchMon"}},
		{"windows counts seconds", "windows", 2, "policy", "shutdown", []string{"/r", "/t", "120", "/c", "policy"}},
	}
	for _, c := range cases {
		name, args := rebootCommand(c.goos, c.delay, c.reason)
		if name != c.wantName || !reflect.DeepEqual(args, c.wantArgs) {
			t.Errorf("%s: got %s %q, want %s %q", c.name, name, args, c.wantName, c.wantArgs)
		}
	}
}

func TestRebootCommandTruncatesLongWindowsComment(t *testing.T) {
	t.Parallel()
	_, args := rebootCommand("windows", 1, strings.Repeat("a", 900))
	if got := len(args[len(args)-1]); got != 500 {
		t.Errorf("comment length = %d, want 500", got)
	}
}
