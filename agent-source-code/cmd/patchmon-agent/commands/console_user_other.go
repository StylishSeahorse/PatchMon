//go:build !darwin

package commands

// getConsoleUser is the non-Darwin counterpart of the implementation in
// sysproc_darwin.go. Its only caller is runPatchBrew, which lives in the
// unconstrained serve.go but is reachable only when the agent reports the
// "brew" package manager — i.e. on macOS. Without this stub the agent fails
// to compile for every other GOOS.
func getConsoleUser() string {
	return ""
}
