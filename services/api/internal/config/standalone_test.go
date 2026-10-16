package config

import "testing"

func TestIsStandalone(t *testing.T) {
	cases := map[string]bool{"": true, "standalone": true, "saas": false}
	for mode, want := range cases {
		c := &Config{Mode: mode}
		if got := c.IsStandalone(); got != want {
			t.Errorf("Mode %q: IsStandalone() = %v, want %v", mode, got, want)
		}
	}
}
