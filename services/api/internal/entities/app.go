package entities

// AppStatus represents the lifecycle status of an app.
type AppStatus string

const (
	AppStatusPending   AppStatus = "pending"
	AppStatusBuilding  AppStatus = "building"
	AppStatusDeploying AppStatus = "deploying"
	AppStatusRunning   AppStatus = "running"
	AppStatusSleeping  AppStatus = "sleeping"
	AppStatusFailed    AppStatus = "failed"
	AppStatusSuspended AppStatus = "suspended"
	AppStatusStopped   AppStatus = "stopped"
)

// DeploySource indicates how the app is deployed.
type DeploySource string

const (
	DeploySourceGit   DeploySource = "git"
	DeploySourceImage DeploySource = "image"
)

// AppType classifies the workload kind.
type AppType string

const (
	AppTypeWeb    AppType = "web"
	AppTypeWorker AppType = "worker"
	AppTypeCron   AppType = "cron"
)

// AppExposure controls how the app is exposed through the gateway.
type AppExposure string

const (
	ExposurePublic    AppExposure = "public"    // Frontend: APISIX passthrough, no auth
	ExposureProtected AppExposure = "protected" // API Service: APISIX + jwt-auth plugin
)

// Framework represents the detected framework type.
type Framework string

const (
	FrameworkNextJS     Framework = "nextjs"
	FrameworkGo         Framework = "go"
	FrameworkPython     Framework = "python"
	FrameworkDjango     Framework = "django"
	FrameworkRails      Framework = "rails"
	FrameworkFlask      Framework = "flask"
	FrameworkExpress    Framework = "express"
	FrameworkStatic     Framework = "static"
	FrameworkDockerfile Framework = "dockerfile"
	FrameworkUnknown    Framework = "unknown"
)

// App represents a user-deployed application on the platform.
type App struct {
	ID               string       `json:"id"`
	UserID           string       `json:"user_id"`
	ProjectID        string       `json:"project_id"`
	EnvironmentID    string       `json:"environment_id,omitempty"`
	Name             string       `json:"name"`
	DeploySource     DeploySource `json:"deploy_source"`
	RepoURL          string       `json:"repo_url"`
	Branch           string       `json:"branch"`
	ImageURL         string       `json:"image_url"`
	RegistryUser     string       `json:"registry_username,omitempty"`
	RegistryPassword string       `json:"registry_password,omitempty"`
	Framework        Framework    `json:"framework"`
	Status           AppStatus    `json:"status"`
	Subdomain        string       `json:"subdomain"`
	Port             int          `json:"port"`
	AppType          AppType      `json:"app_type"`
	Command          string       `json:"command,omitempty"`
	CronSchedule     string       `json:"cron_schedule,omitempty"`
	Exposure         AppExposure  `json:"exposure"`
	AutoGatewayID    string       `json:"auto_gateway_id,omitempty"`
	// Replicas is the desired number of pod replicas. Defaults to 1.
	Replicas int `json:"replicas"`
	// HealthCheckPath is the HTTP path for K8s liveness/readiness probes. Defaults to "/".
	HealthCheckPath string `json:"health_check_path"`
	// DependsOn lists the K8s service names (subdomains) of apps this app depends on.
	// Used to generate init containers that wait for dependencies before starting.
	DependsOn []string `json:"depends_on,omitempty"`
	Timestamps
}
