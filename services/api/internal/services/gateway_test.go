package services

import (
	"encoding/json"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// --- slugify tests ---

func TestSlugify_BasicName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"My Gateway", "my-gateway"},
		{"Hello World API", "hello-world-api"},
		{"simple", "simple"},
		{"UPPER", "upper"},
		{"with  spaces", "with-spaces"},
		{"special!@#chars", "special-chars"},
		{"---dashes---", "dashes"},
		{"", "gateway"},
		{"   ", "gateway"},
		{"a--b--c", "a-b-c"},
	}

	for _, tc := range cases {
		got := slugify(tc.input)
		if got != tc.expected {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// --- validatePlugins tests ---

func TestValidatePlugins_AllowedPlugins(t *testing.T) {
	plugins := []entities.GatewayRoutePlugin{
		{Name: "cors", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "limit-count", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "jwt-auth", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "key-auth", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "ip-restriction", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "proxy-rewrite", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "request-id", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "openid-connect", Enable: true, Config: json.RawMessage(`{}`)},
	}

	err := validatePlugins(plugins)
	if err != nil {
		t.Errorf("Expected all plugins to be allowed, got: %v", err)
	}
}

func TestValidatePlugins_DisallowedPlugin(t *testing.T) {
	plugins := []entities.GatewayRoutePlugin{
		{Name: "cors", Enable: true, Config: json.RawMessage(`{}`)},
		{Name: "dangerous-plugin", Enable: true, Config: json.RawMessage(`{}`)},
	}

	err := validatePlugins(plugins)
	if err == nil {
		t.Error("Expected error for disallowed plugin")
	}
}

func TestValidatePlugins_Empty(t *testing.T) {
	err := validatePlugins(nil)
	if err != nil {
		t.Errorf("Expected no error for empty plugins, got: %v", err)
	}

	err = validatePlugins([]entities.GatewayRoutePlugin{})
	if err != nil {
		t.Errorf("Expected no error for empty plugins slice, got: %v", err)
	}
}

func TestValidatePlugins_SingleDisallowed(t *testing.T) {
	plugins := []entities.GatewayRoutePlugin{
		{Name: "exec", Enable: true, Config: json.RawMessage(`{}`)},
	}

	err := validatePlugins(plugins)
	if err == nil {
		t.Error("Expected error for single disallowed plugin")
	}
}
