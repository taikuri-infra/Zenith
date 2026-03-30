package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yml structure (minimal — only what we need).
type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService represents a single service in a docker-compose file.
type ComposeService struct {
	Image       string      `yaml:"image"`
	Build       interface{} `yaml:"build"`       // string or map
	Ports       []string    `yaml:"ports"`
	Environment interface{} `yaml:"environment"` // map or list
	DependsOn   interface{} `yaml:"depends_on"`  // list or map
	Volumes     []string    `yaml:"volumes"`
	Command     interface{} `yaml:"command"`
	Restart     string      `yaml:"restart"`
}

// ParsedCompose is the result of parsing a docker-compose file.
type ParsedCompose struct {
	Valid           bool            `json:"valid"`
	Services        []ParsedService `json:"services"`
	ManagedServices []ParsedManaged `json:"managed_services"`
	Warnings        []string        `json:"warnings"`
	Errors          []string        `json:"errors"`
}

// ParsedService represents a detected app service.
type ParsedService struct {
	Name         string         `json:"name"`
	BuildContext string         `json:"build_context,omitempty"`
	Image        string         `json:"image,omitempty"`
	Port         int            `json:"port"`
	IsPublic     bool           `json:"is_public"`
	URL          string         `json:"url,omitempty"`
	EnvVars      []ParsedEnvVar `json:"env_vars"`
	DependsOn    []string       `json:"depends_on"`
	// Command is the compose `command:` override (maps to K8s container args).
	Command string `json:"command,omitempty"`
	// HasVolumes is true if the service declares any volumes (data will be ephemeral).
	HasVolumes bool `json:"has_volumes,omitempty"`
}

// ParsedEnvVar represents an environment variable translation.
type ParsedEnvVar struct {
	Key      string `json:"key"`
	Original string `json:"original"`
	Zenith   string `json:"zenith"`
}

// ParsedManaged represents a detected managed service (database/cache).
type ParsedManaged struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	DetectedFrom string `json:"detected_from"`
}

// managedImages maps docker image names to platform service types.
var managedImages = map[string]entities.ServiceType{
	"postgres":      entities.ServiceTypePostgreSQL,
	"postgresql":    entities.ServiceTypePostgreSQL,
	"redis":         entities.ServiceTypeRedis,
	"valkey":        entities.ServiceTypeRedis,
	"mysql":         entities.ServiceTypeMySQL,
	"mariadb":       entities.ServiceTypeMySQL,
	"mongo":         entities.ServiceTypeMongoDB,
	"mongodb":       entities.ServiceTypeMongoDB,
	"rabbitmq":      entities.ServiceTypeRabbitMQ,
	"elasticsearch": entities.ServiceTypeElasticsearch,
	"opensearch":    entities.ServiceTypeOpenSearch,
	"kafka":         entities.ServiceTypeKafka,
	"nats":          entities.ServiceTypeNATS,
	"memcached":     entities.ServiceTypeMemcached,
	"minio":         entities.ServiceTypeMinIO,
	"clickhouse":    entities.ServiceTypeClickHouse,
	"zookeeper":     entities.ServiceTypeKafka, // Kafka dependency — treat as infrastructure
}

// serviceURLPattern matches URL-style service references like http://api:8080 or postgres://db:5432
var serviceURLPattern = regexp.MustCompile(`(https?|postgresql|postgres|redis|amqp|amqps|mysql|mongodb|mongodb\+srv)://([a-zA-Z0-9_-]+):(\d+)`)

// plainHostPattern matches plain hostname env var values like DB_HOST=postgres or REDIS_HOST=cache
var plainHostPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)$`)

// ParseCompose parses a docker-compose.yml content and detects services and managed services.
// baseDomain is the platform base domain (e.g. "apps.stage.freezenith.com") used to generate
// preview URLs for public services. Pass "" to omit URL generation.
func ParseCompose(content string, projectSlug, namespace, baseDomain string) (*ParsedCompose, error) {
	result := &ParsedCompose{
		Valid:           true,
		Services:        make([]ParsedService, 0),
		ManagedServices: make([]ParsedManaged, 0),
		Warnings:        make([]string, 0),
		Errors:          make([]string, 0),
	}

	// Layer 1: YAML parse
	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(content), &compose); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid YAML: %s", err.Error()))
		return result, nil
	}

	if len(compose.Services) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "no services found in compose file")
		return result, nil
	}

	// Build a set of service names for URL translation
	serviceNames := make(map[string]bool)
	for name := range compose.Services {
		serviceNames[name] = true
	}

	// Classify each service
	for name, svc := range compose.Services {
		imageBase := extractImageBase(svc.Image)

		// Check if this is a managed service (database/cache)
		if st, ok := managedImages[imageBase]; ok {
			version := detectVersion(svc.Image)
			result.ManagedServices = append(result.ManagedServices, ParsedManaged{
				Name:         name,
				Type:         string(st),
				Version:      version,
				DetectedFrom: svc.Image,
			})
			continue
		}

		// This is an app service
		port := extractPort(svc.Ports)
		envMap := parseEnvironment(svc.Environment)
		dependsOn := parseDependsOn(svc.DependsOn)

		var envVars []ParsedEnvVar
		for key, val := range envMap {
			zenithVal := translateServiceURL(val, projectSlug, namespace, serviceNames)
			// Also translate plain hostname values like DB_HOST=postgres → DB_HOST=postgres-slug.ns.svc
			if zenithVal == val && projectSlug != "" && namespace != "" {
				if plainHostPattern.MatchString(val) && serviceNames[val] {
					zenithVal = fmt.Sprintf("%s-%s.%s.svc", val, projectSlug, namespace)
				}
			}
			envVars = append(envVars, ParsedEnvVar{
				Key:      key,
				Original: val,
				Zenith:   zenithVal,
			})
		}

		// Only mark as public if it looks like a frontend service.
		// Services named "api", "backend", "worker", "server", "grpc" etc. stay internal.
		isPublic := port > 0 && isFrontendService(name, svc.Image, port)
		var url string
		if isPublic && projectSlug != "" && baseDomain != "" {
			url = fmt.Sprintf("https://%s-%s.%s", name, projectSlug, baseDomain)
		}

		buildCtx := ""
		if svc.Build != nil {
			switch b := svc.Build.(type) {
			case string:
				buildCtx = b
			case map[string]interface{}:
				if ctx, ok := b["context"].(string); ok {
					buildCtx = ctx
				}
			}
		}

		// Extract command (compose `command:` overrides the image CMD → K8s args).
		command := extractCommand(svc.Command)

		result.Services = append(result.Services, ParsedService{
			Name:         name,
			BuildContext: buildCtx,
			Image:        svc.Image,
			Port:         port,
			IsPublic:     isPublic,
			URL:          url,
			EnvVars:      envVars,
			DependsOn:    dependsOn,
			Command:      command,
			HasVolumes:   len(svc.Volumes) > 0,
		})
	}

	// Generate warnings
	if len(result.Services) == 0 {
		result.Warnings = append(result.Warnings, "no app services detected — all services appear to be databases/caches")
	}

	for _, svc := range result.Services {
		for _, ev := range svc.EnvVars {
			if containsHardcodedPassword(ev.Original) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("service '%s': env var '%s' may contain a hardcoded password", svc.Name, ev.Key))
			}
		}
		if svc.Port == 0 && svc.BuildContext != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("service '%s': no port detected — you may need to configure the port manually", svc.Name))
		}
		if svc.HasVolumes {
			result.Warnings = append(result.Warnings, fmt.Sprintf("service '%s': declares volumes — data will be EPHEMERAL on Zenith (no persistent storage by default)", svc.Name))
		}
	}

	return result, nil
}

// extractImageBase extracts the base image name (without tag/registry).
func extractImageBase(image string) string {
	// Remove registry prefix (e.g., docker.io/library/)
	parts := strings.Split(image, "/")
	base := parts[len(parts)-1]
	// Remove tag
	if idx := strings.Index(base, ":"); idx > 0 {
		base = base[:idx]
	}
	return strings.ToLower(base)
}

// detectVersion extracts version from an image string (e.g., "postgres:16" → "16").
func detectVersion(image string) string {
	if idx := strings.LastIndex(image, ":"); idx > 0 {
		tag := image[idx+1:]
		if tag != "latest" && tag != "" {
			return tag
		}
	}
	return "latest"
}

// extractPort parses the first port mapping (e.g., "3000:3000" → 3000).
func extractPort(ports []string) int {
	if len(ports) == 0 {
		return 0
	}
	// Take the first port mapping
	p := ports[0]
	// Handle "host:container" format
	parts := strings.Split(p, ":")
	if len(parts) >= 2 {
		port, err := strconv.Atoi(strings.Split(parts[1], "/")[0])
		if err == nil {
			return port
		}
	}
	// Handle single port
	port, err := strconv.Atoi(strings.Split(p, "/")[0])
	if err == nil {
		return port
	}
	return 0
}

// parseEnvironment handles both map and list formats for environment variables.
func parseEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)
	if env == nil {
		return result
	}

	switch e := env.(type) {
	case map[string]interface{}:
		for k, v := range e {
			result[k] = fmt.Sprintf("%v", v)
		}
	case []interface{}:
		for _, item := range e {
			s, ok := item.(string)
			if !ok {
				continue
			}
			if idx := strings.Index(s, "="); idx > 0 {
				result[s[:idx]] = s[idx+1:]
			} else {
				result[s] = ""
			}
		}
	}
	return result
}

// parseDependsOn handles both list and map formats for depends_on.
func parseDependsOn(dep interface{}) []string {
	if dep == nil {
		return nil
	}

	switch d := dep.(type) {
	case []interface{}:
		var result []string
		for _, item := range d {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case map[string]interface{}:
		var result []string
		for k := range d {
			result = append(result, k)
		}
		return result
	}
	return nil
}

// translateServiceURL replaces compose service references with K8s DNS names.
// e.g., "http://api:8080" → "http://api-myproject.zenith-apps.svc:8080"
func translateServiceURL(value, projectSlug, namespace string, serviceNames map[string]bool) string {
	return serviceURLPattern.ReplaceAllStringFunc(value, func(match string) string {
		submatches := serviceURLPattern.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}
		scheme := submatches[1]
		host := submatches[2]
		port := submatches[3]

		if serviceNames[host] {
			k8sHost := fmt.Sprintf("%s-%s.%s.svc", host, projectSlug, namespace)
			return fmt.Sprintf("%s://%s:%s", scheme, k8sHost, port)
		}
		return match
	})
}

var hardcodedPasswordPattern = regexp.MustCompile(`(?i)(password|passwd|secret)=\S+`)

// containsHardcodedPassword checks if a value looks like it contains a hardcoded password.
func containsHardcodedPassword(value string) bool {
	return hardcodedPasswordPattern.MatchString(value)
}

// backendNames lists service name patterns that should NOT be public.
var backendNames = []string{"api", "backend", "server", "worker", "grpc", "queue", "cron", "scheduler", "internal"}

// frontendImages lists image names that are always public.
var frontendImages = []string{"nginx", "httpd", "apache", "caddy", "traefik"}

// extractCommand converts a compose `command:` value (string or list) to a single string.
// The compose `command:` overrides the image CMD; in K8s this maps to container args.
func extractCommand(cmd interface{}) string {
	if cmd == nil {
		return ""
	}
	switch c := cmd.(type) {
	case string:
		return c
	case []interface{}:
		parts := make([]string, 0, len(c))
		for _, v := range c {
			if s, ok := v.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

// isFrontendService determines if a service should be publicly exposed.
// Public: services on port 80/443/3000 with frontend-like names/images.
// Internal: services named api/backend/worker/server, or on non-web ports.
func isFrontendService(name, image string, port int) bool {
	nameLower := strings.ToLower(name)
	imageLower := strings.ToLower(image)

	// Explicit backend names → internal
	for _, bn := range backendNames {
		if strings.Contains(nameLower, bn) {
			return false
		}
	}

	// Known frontend images → public
	for _, fi := range frontendImages {
		if strings.Contains(imageLower, fi) {
			return true
		}
	}

	// Common frontend ports → public (NOT 8080 — too ambiguous, could be backend API)
	if port == 80 || port == 443 || port == 3000 {
		return true
	}

	// Default: internal (safe default — user can change in dashboard)
	return false
}
