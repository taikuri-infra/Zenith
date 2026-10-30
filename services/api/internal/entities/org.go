package entities

import (
	"regexp"
	"strings"
)

// LegacyAppsNamespace is where workloads ran before Organizations existed.
// Each project's migrated default Org keeps living here.
const LegacyAppsNamespace = "zenith-apps"

// Org is an isolation boundary inside a Project: it owns one Kubernetes namespace.
type Org struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Namespace string `json:"namespace"`
	IsDefault bool   `json:"is_default"`
	Timestamps
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// OrgSlug turns a display name into a DNS-label-safe slug.
func OrgSlug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 30 {
		s = strings.Trim(s[:30], "-")
	}
	return s
}

// OrgNamespace derives the namespace of a new Org: zo-<first 8 of project id>-<slug>.
func OrgNamespace(projectID, slug string) string {
	id := strings.ToLower(strings.TrimPrefix(projectID, "proj-"))
	id = nonSlug.ReplaceAllString(id, "")
	if len(id) > 8 {
		id = id[:8]
	}
	return "zo-" + id + "-" + slug
}
