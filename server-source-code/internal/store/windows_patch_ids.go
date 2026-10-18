package store

import (
	"regexp"
	"strings"

	"github.com/PatchMon/PatchMon/server-source-code/internal/util"
)

// WUA update types the agent reports in wuaType.
const (
	WUAUpdateTypeSoftware = "Software"
	WUAUpdateTypeDriver   = "Driver"
	WUAUpdateTypeFeature  = "FeatureUpgrade"
)

var (
	wingetIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+\-]{0,127}$`)
	wuaGUIDPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// IsValidWingetID reports whether id is safe to hand to the agent as a winget --id value.
func IsValidWingetID(id string) bool {
	return wingetIDPattern.MatchString(id)
}

// IsValidWUAGuid reports whether id is a Windows Update UpdateID.
func IsValidWUAGuid(id string) bool {
	return wuaGUIDPattern.MatchString(id)
}

// IsWindowsOSType applies the same Windows host test as the frontend.
func IsWindowsOSType(osType string) bool {
	return strings.Contains(strings.ToLower(osType), "windows")
}

// MinWindowsPatchAgentVersion is the oldest agent that may be sent a Windows patch run.
const MinWindowsPatchAgentVersion = "2.1.4"

// Lowest semver pre-release of the minimum, so 2.1.4-rc.N builds pass the gate.
const minWindowsPatchAgentFloor = MinWindowsPatchAgentVersion + "-0"

// WindowsAgentCanPatch reports whether the agent is new enough to be sent a Windows patch run.
func WindowsAgentCanPatch(agentVersion *string) bool {
	v := ""
	if agentVersion != nil {
		v = *agentVersion
	}
	return v != "" && util.CompareVersions(v, minWindowsPatchAgentFloor) >= 0
}
