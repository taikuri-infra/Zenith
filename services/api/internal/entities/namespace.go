package entities

import (
	"regexp"
	"strings"
)

// LegacyAppsNamespace is where workloads ran before projects had their own
// namespace; existing projects keep using it so nothing moves.
const LegacyAppsNamespace = "zenith-apps"

var nsUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// ProjectNamespace derives the namespace of a new project: zp-<first 8 of id>-<slug>.
func ProjectNamespace(projectID, slug string) string {
	id := nsUnsafe.ReplaceAllString(strings.ToLower(projectID), "")
	if len(id) > 8 {
		id = id[:8]
	}
	s := strings.Trim(nsUnsafe.ReplaceAllString(strings.ToLower(slug), "-"), "-")
	if len(s) > 30 {
		s = strings.Trim(s[:30], "-")
	}
	if s == "" {
		return "zp-" + id
	}
	return "zp-" + id + "-" + s
}
