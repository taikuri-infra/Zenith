// Package hetzner provides backward-compatible aliases for the hetznerclient adapter.
// New code should import "github.com/dotechhq/zenith/services/api/internal/adapters/hetznerclient" directly.
package hetzner

import "github.com/dotechhq/zenith/services/api/internal/adapters/hetznerclient"

// Type aliases
type HetznerAPI = hetznerclient.HetznerAPI
type ServerResult = hetznerclient.ServerResult
type Client = hetznerclient.Client

// Constructor aliases
var NewClient = hetznerclient.NewClient
