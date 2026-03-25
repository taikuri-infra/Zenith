package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// ImageVerifyHandler verifies whether Docker images are accessible before deploy.
type ImageVerifyHandler struct {
	projectRepo  ports.ProjectRepository
	registryHost string // e.g. "registry.stage.freezenith.com"
	registryUser string // robot account for internal registry
	registryPass string
}

// NewImageVerifyHandler creates a new ImageVerifyHandler.
func NewImageVerifyHandler(projectRepo ports.ProjectRepository, registryHost, registryUser, registryPass string) *ImageVerifyHandler {
	return &ImageVerifyHandler{
		projectRepo:  projectRepo,
		registryHost: registryHost,
		registryUser: registryUser,
		registryPass: registryPass,
	}
}

type verifyImageInput struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type verifyImagesRequest struct {
	Images []verifyImageInput `json:"images"`
}

type ImageVerifyResult struct {
	Name      string `json:"name"`
	Image     string `json:"image"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
}

type VerifyImagesResponse struct {
	AllReady bool                `json:"all_ready"`
	Results  []ImageVerifyResult `json:"results"`
}

// VerifyImages handles POST /projects/:projectId/verify-images
// Checks each provided image against its registry to confirm it exists and is pullable.
func (h *ImageVerifyHandler) VerifyImages(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	projectID := c.Params("projectId")
	project, err := h.projectRepo.GetProject(c.Context(), projectID)
	if err != nil {
		return NewNotFound("project not found")
	}
	if project.UserID != userID {
		return NewForbidden("not your project")
	}

	var req verifyImagesRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if len(req.Images) == 0 {
		return c.JSON(VerifyImagesResponse{AllReady: true, Results: []ImageVerifyResult{}})
	}

	resp := VerifyImagesResponse{AllReady: true, Results: make([]ImageVerifyResult, 0, len(req.Images))}

	for _, img := range req.Images {
		result := ImageVerifyResult{Name: img.Name, Image: img.Image}

		if h.registryHost != "" && strings.HasPrefix(img.Image, h.registryHost+"/") {
			// Zenith internal registry — use robot credentials
			if h.registryUser != "" {
				result.Reachable, result.Error = checkManifest(c.Context(), img.Image, h.registryUser, h.registryPass)
			} else {
				// Credentials not configured: can't verify, let it through with warning
				result.Reachable = true
				result.Error = "cannot verify (registry credentials not configured)"
			}
		} else {
			// External registry — try without credentials (public image check)
			result.Reachable, result.Error = checkManifest(c.Context(), img.Image, "", "")
		}

		if !result.Reachable {
			resp.AllReady = false
		}
		resp.Results = append(resp.Results, result)
	}

	return c.JSON(resp)
}

// checkManifest does a Docker Registry v2 manifest HEAD request to verify an image exists.
// For Docker Hub and other public registries, a 401 (auth challenge) means the image EXISTS
// but requires a token — we treat that as "reachable" since the kubelet will pull it fine.
func checkManifest(ctx context.Context, image, user, pass string) (bool, string) {
	registry, repo, tag := splitImageRef(image)

	var manifestURL string
	if registry == "docker.io" {
		manifestURL = fmt.Sprintf("https://index.docker.io/v2/%s/manifests/%s", repo, tag)
	} else {
		manifestURL = fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repo, tag)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return false, "failed to build request"
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "registry unreachable: " + trimErr(err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, ""
	case http.StatusUnauthorized:
		// For external registries (Docker Hub, GHCR): 401 on HEAD means the image exists
		// but needs a token. Kubelet can pull it as long as the image name is correct.
		// For internal registry with credentials: 401 means wrong credentials.
		if user != "" {
			return false, "authentication failed — check registry credentials"
		}
		return true, "" // external image exists, needs pull credentials at runtime
	case http.StatusForbidden:
		if user != "" {
			return false, "access denied — check registry permissions"
		}
		return true, "" // private image, treat as exists
	case http.StatusNotFound:
		return false, fmt.Sprintf("image not found — check the name and tag %q", tag)
	default:
		return false, fmt.Sprintf("registry returned HTTP %d", resp.StatusCode)
	}
}

// splitImageRef parses "registry/repo:tag" into its three parts.
// Handles Docker Hub shorthand (nginx → docker.io/library/nginx:latest).
func splitImageRef(image string) (registry, repo, tag string) {
	tag = "latest"

	// Extract tag (everything after last ":" that doesn't look like a port)
	if idx := strings.LastIndex(image, ":"); idx > 0 {
		candidate := image[idx+1:]
		if !strings.Contains(candidate, "/") {
			tag = candidate
			image = image[:idx]
		}
	}

	parts := strings.SplitN(image, "/", 2)
	if len(parts) == 1 {
		// e.g. "nginx" → Docker Hub official library
		return "docker.io", "library/" + parts[0], tag
	}

	// First segment is a registry host if it contains a dot or colon, or is "localhost"
	if strings.ContainsAny(parts[0], ".:") || parts[0] == "localhost" {
		return parts[0], parts[1], tag
	}

	// Docker Hub with namespace: "myuser/myapp"
	return "docker.io", image, tag
}

func trimErr(err error) string {
	msg := err.Error()
	if len(msg) > 80 {
		return msg[:80] + "..."
	}
	return msg
}
