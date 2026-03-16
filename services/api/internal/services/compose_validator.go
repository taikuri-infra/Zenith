package services

import "fmt"

// ValidateCompose performs Layer 2 validation on a parsed compose result.
// Returns a list of warning/error strings. An empty list means no issues found.
func ValidateCompose(parsed *ParsedCompose) []string {
	var issues []string

	// Must have at least one app service
	if len(parsed.Services) == 0 {
		issues = append(issues, "error: no app services detected — your compose file only contains databases/caches")
	}

	// Check for duplicate service names
	seen := make(map[string]bool)
	for _, svc := range parsed.Services {
		if seen[svc.Name] {
			issues = append(issues, fmt.Sprintf("error: duplicate service name '%s'", svc.Name))
		}
		seen[svc.Name] = true
	}
	for _, ms := range parsed.ManagedServices {
		if seen[ms.Name] {
			issues = append(issues, fmt.Sprintf("error: duplicate service name '%s'", ms.Name))
		}
		seen[ms.Name] = true
	}

	// Validate ports
	for _, svc := range parsed.Services {
		if svc.Port > 65535 {
			issues = append(issues, fmt.Sprintf("error: service '%s' has invalid port %d", svc.Name, svc.Port))
		}
	}

	// Check depends_on references exist
	allNames := make(map[string]bool)
	for _, svc := range parsed.Services {
		allNames[svc.Name] = true
	}
	for _, ms := range parsed.ManagedServices {
		allNames[ms.Name] = true
	}
	for _, svc := range parsed.Services {
		for _, dep := range svc.DependsOn {
			if !allNames[dep] {
				issues = append(issues, fmt.Sprintf("warning: service '%s' depends on '%s' which was not found", svc.Name, dep))
			}
		}
	}

	// Warning for services without a port or build context
	for _, svc := range parsed.Services {
		if svc.Port == 0 && svc.BuildContext == "" && svc.Image == "" {
			issues = append(issues, fmt.Sprintf("warning: service '%s' has no port, build context, or image", svc.Name))
		}
	}

	return issues
}
