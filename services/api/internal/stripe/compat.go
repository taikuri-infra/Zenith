// Package stripe provides backward-compatible aliases for the stripeclient adapter.
// New code should import "github.com/dotechhq/zenith/services/api/internal/adapters/stripeclient" directly.
package stripe

import "github.com/dotechhq/zenith/services/api/internal/adapters/stripeclient"

// Type aliases
type CheckoutParams = stripeclient.CheckoutParams
type CheckoutResult = stripeclient.CheckoutResult
type PortalResult = stripeclient.PortalResult
type SubscriptionResult = stripeclient.SubscriptionResult
type StripeAPI = stripeclient.StripeAPI
type Client = stripeclient.Client

// Constructor aliases
var NewClient = stripeclient.NewClient
