package deploy

import (
	"os"
	"path/filepath"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// fileMarker maps a file path to a framework.
type fileMarker struct {
	File      string
	Framework entities.Framework
	// Priority is used when multiple markers match. Higher wins.
	Priority int
}

// markers defines the detection rules ordered by specificity.
// When a Dockerfile is present, we always use it (highest priority).
var markers = []fileMarker{
	{File: "Dockerfile", Framework: entities.FrameworkDockerfile, Priority: 100},
	{File: "next.config.js", Framework: entities.FrameworkNextJS, Priority: 20},
	{File: "next.config.ts", Framework: entities.FrameworkNextJS, Priority: 20},
	{File: "next.config.mjs", Framework: entities.FrameworkNextJS, Priority: 20},
	{File: "Gemfile", Framework: entities.FrameworkRails, Priority: 15},
	{File: "manage.py", Framework: entities.FrameworkDjango, Priority: 15},
	{File: "go.mod", Framework: entities.FrameworkGo, Priority: 10},
	{File: "requirements.txt", Framework: entities.FrameworkPython, Priority: 8},
	{File: "Pipfile", Framework: entities.FrameworkPython, Priority: 8},
	{File: "pyproject.toml", Framework: entities.FrameworkPython, Priority: 8},
	{File: "package.json", Framework: entities.FrameworkExpress, Priority: 5},
	{File: "index.html", Framework: entities.FrameworkStatic, Priority: 1},
}

// DetectFramework analyzes a directory and returns the detected framework.
// It checks for known file markers in priority order.
func DetectFramework(repoDir string) entities.Framework {
	best := entities.FrameworkUnknown
	bestPriority := -1

	for _, m := range markers {
		path := filepath.Join(repoDir, m.File)
		if _, err := os.Stat(path); err == nil {
			if m.Priority > bestPriority {
				best = m.Framework
				bestPriority = m.Priority
			}
		}
	}

	// Refine: if package.json exists AND has next dependency, it's Next.js
	if best == entities.FrameworkExpress {
		if hasNextDep(filepath.Join(repoDir, "package.json")) {
			best = entities.FrameworkNextJS
		}
	}

	// Refine: if requirements.txt + manage.py → Django, if flask in requirements → Flask
	if best == entities.FrameworkPython {
		if hasFlaskDep(repoDir) {
			best = entities.FrameworkFlask
		}
	}

	return best
}

// DetectFrameworkFromFiles is a simpler detection that works from a list of filenames
// (useful when we don't have the actual directory, e.g. from GitHub API tree).
func DetectFrameworkFromFiles(files []string) entities.Framework {
	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[filepath.Base(f)] = true
	}

	best := entities.FrameworkUnknown
	bestPriority := -1

	for _, m := range markers {
		if fileSet[m.File] {
			if m.Priority > bestPriority {
				best = m.Framework
				bestPriority = m.Priority
			}
		}
	}

	return best
}

// hasNextDep checks if package.json contains "next" as dependency.
func hasNextDep(packageJSONPath string) bool {
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}
	// Simple string search — avoids full JSON parse for speed
	content := string(data)
	return contains(content, `"next"`)
}

// hasFlaskDep checks if requirements.txt or Pipfile mentions flask.
func hasFlaskDep(repoDir string) bool {
	for _, f := range []string{"requirements.txt", "Pipfile", "pyproject.toml"} {
		data, err := os.ReadFile(filepath.Join(repoDir, f))
		if err != nil {
			continue
		}
		if contains(string(data), "flask") || contains(string(data), "Flask") {
			return true
		}
	}
	return false
}

// contains is a simple substring check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
