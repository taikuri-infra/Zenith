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
	Timestamps
}
