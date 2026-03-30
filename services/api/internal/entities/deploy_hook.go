package entities

// DeployHookType classifies the hook execution mechanism.
type DeployHookType string

const (
	DeployHookHTTP    DeployHookType = "http"    // POST payload to URL
	DeployHookCommand DeployHookType = "command" // exec in app container
)

// DeployHook is a user-defined action that runs after a successful deployment.
type DeployHook struct {
	ID      string         `json:"id"`
	AppID   string         `json:"app_id"`
	Name    string         `json:"name"`
	Type    DeployHookType `json:"type"`
	URL     string         `json:"url,omitempty"`     // for HTTP hooks
	Command string         `json:"command,omitempty"` // for command hooks
	Order   int            `json:"order"`             // execution sequence (lower = first)
	Active  bool           `json:"active"`
	Timestamps
}
