// Package k8s provides backward-compatible aliases for the k8sclient adapter.
// New code should import "github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient" directly.
package k8s

import "github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"

// Type aliases
type CRDObject = k8sclient.CRDObject
type ObjectMeta = k8sclient.ObjectMeta
type JobObject = k8sclient.JobObject
type Client = k8sclient.Client
type MemoryClient = k8sclient.MemoryClient
type RealClient = k8sclient.RealClient

// Constructor aliases
var NewMemoryClient = k8sclient.NewMemoryClient
var NewRealClient = k8sclient.NewRealClient
