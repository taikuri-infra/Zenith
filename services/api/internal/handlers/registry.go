package handlers

import (
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/harborclient"
	"github.com/gofiber/fiber/v2"
)

// RegistryHandler provides endpoints for browsing Harbor repositories and scan results.
type RegistryHandler struct {
	harbor  *harborclient.Client
	project string // Harbor project name (e.g. "zenith-stage")
}

// NewRegistryHandler creates a new RegistryHandler.
func NewRegistryHandler(harbor *harborclient.Client, project string) *RegistryHandler {
	return &RegistryHandler{harbor: harbor, project: project}
}

type registryRepoResponse struct {
	Name          string             `json:"name"`
	ArtifactCount int                `json:"artifact_count"`
	LastPushed    string             `json:"last_pushed"`
	Artifacts     []registryArtifact `json:"artifacts,omitempty"`
	Scan          *registryScanInfo  `json:"scan,omitempty"`
}

type registryArtifact struct {
	Tag      string          `json:"tag"`
	Digest   string          `json:"digest"`
	Size     string          `json:"size"`
	Pushed   string          `json:"pushed"`
	Status   string          `json:"status"` // "passed", "warning", "failed", "pending"
	Critical int             `json:"critical"`
	High     int             `json:"high"`
	Medium   int             `json:"medium"`
}

type registryScanInfo struct {
	Passed  int `json:"passed"`
	Warning int `json:"warning"`
	Failed  int `json:"failed"`
	Total   int `json:"total"`
}

// ListRepositories returns all repositories with their artifacts and scan status.
// GET /api/v1/registry/repos
func (h *RegistryHandler) ListRepositories(c *fiber.Ctx) error {
	if h.harbor == nil {
		return c.JSON([]registryRepoResponse{})
	}

	repos, err := h.harbor.ListRepositories(c.Context(), h.project)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "failed to list repositories: "+err.Error())
	}

	result := make([]registryRepoResponse, 0, len(repos))
	for _, repo := range repos {
		// Strip project prefix from name (e.g. "zenith-stage/myapp" -> "myapp")
		name := repo.Name
		if idx := strings.Index(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}

		entry := registryRepoResponse{
			Name:          name,
			ArtifactCount: repo.ArtifactCount,
			LastPushed:    repo.UpdateTime,
		}

		// Fetch artifacts with scan overview
		artifacts, err := h.harbor.ListArtifacts(c.Context(), h.project, name, true)
		if err == nil {
			scan := &registryScanInfo{}
			for _, art := range artifacts {
				tagName := ""
				if len(art.Tags) > 0 {
					tagName = art.Tags[0].Name
				}
				if tagName == "" {
					tagName = art.Digest[:12]
				}

				artResp := registryArtifact{
					Tag:    tagName,
					Digest: art.Digest,
					Size:   formatBytes(art.Size),
					Pushed: art.PushTime.Format("2006-01-02T15:04:05Z"),
				}

				// Extract scan results
				for _, overview := range art.ScanOverview {
					if overview == nil {
						continue
					}
					switch overview.ScanStatus {
					case "Success":
						if overview.Summary != nil {
							artResp.Critical = overview.Summary.Critical
							artResp.High = overview.Summary.High
							artResp.Medium = overview.Summary.Medium
							if overview.Summary.Critical > 0 || overview.Summary.High > 0 {
								artResp.Status = "failed"
								scan.Failed++
							} else if overview.Summary.Medium > 0 {
								artResp.Status = "warning"
								scan.Warning++
							} else {
								artResp.Status = "passed"
								scan.Passed++
							}
						} else {
							artResp.Status = "passed"
							scan.Passed++
						}
					case "Running", "Pending":
						artResp.Status = "pending"
					default:
						artResp.Status = "warning"
						scan.Warning++
					}
					scan.Total++
					break // only process first scan overview
				}
				if artResp.Status == "" {
					artResp.Status = "pending"
				}

				entry.Artifacts = append(entry.Artifacts, artResp)
			}
			entry.Scan = scan
		}

		result = append(result, entry)
	}

	return c.JSON(result)
}

// GetRepository returns a single repository with all artifacts and scan details.
// GET /api/v1/registry/repos/:name
func (h *RegistryHandler) GetRepository(c *fiber.Ctx) error {
	name := c.Params("name")
	if h.harbor == nil {
		return fiber.NewError(fiber.StatusNotFound, "registry not configured")
	}

	artifacts, err := h.harbor.ListArtifacts(c.Context(), h.project, name, true)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "repository not found")
	}

	entry := registryRepoResponse{
		Name:          name,
		ArtifactCount: len(artifacts),
	}

	scan := &registryScanInfo{}
	for _, art := range artifacts {
		tagName := ""
		if len(art.Tags) > 0 {
			tagName = art.Tags[0].Name
		}
		if tagName == "" {
			tagName = art.Digest[:12]
		}

		artResp := registryArtifact{
			Tag:    tagName,
			Digest: art.Digest,
			Size:   formatBytes(art.Size),
			Pushed: art.PushTime.Format("2006-01-02T15:04:05Z"),
		}

		for _, overview := range art.ScanOverview {
			if overview == nil {
				continue
			}
			switch overview.ScanStatus {
			case "Success":
				if overview.Summary != nil {
					artResp.Critical = overview.Summary.Critical
					artResp.High = overview.Summary.High
					artResp.Medium = overview.Summary.Medium
					if overview.Summary.Critical > 0 || overview.Summary.High > 0 {
						artResp.Status = "failed"
						scan.Failed++
					} else if overview.Summary.Medium > 0 {
						artResp.Status = "warning"
						scan.Warning++
					} else {
						artResp.Status = "passed"
						scan.Passed++
					}
				} else {
					artResp.Status = "passed"
					scan.Passed++
				}
			default:
				artResp.Status = "pending"
			}
			scan.Total++
			break
		}
		if artResp.Status == "" {
			artResp.Status = "pending"
		}

		entry.Artifacts = append(entry.Artifacts, artResp)
	}
	entry.Scan = scan

	return c.JSON(entry)
}

func formatBytes(b int64) string {
	const mb = 1024 * 1024
	const gb = 1024 * mb
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	}
	return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
}
