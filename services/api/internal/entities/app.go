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
	AppStatusStopped   AppStatus = "stopped"
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
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	RepoURL   string    `json:"repo_url"`
	Branch    string    `json:"branch"`
	Framework Framework `json:"framework"`
	Status    AppStatus `json:"status"`
	Subdomain string    `json:"subdomain"`
	Port      int       `json:"port"`
	Timestamps
}
