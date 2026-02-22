// Package capi provides backward-compatible aliases for the capiclient adapter.
// New code should import "github.com/dotechhq/zenith/services/api/internal/adapters/capiclient" directly.
package capi

import "github.com/dotechhq/zenith/services/api/internal/adapters/capiclient"

// Constant aliases
const (
	CAPINamespace = capiclient.CAPINamespace
	KindCluster   = capiclient.KindCluster
)

// Type aliases
type Client = capiclient.Client

// Constructor aliases
var NewClient = capiclient.NewClient
