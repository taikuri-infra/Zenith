package entities

import (
	"strings"
	"testing"
)

func TestProjectNamespace(t *testing.T) {
	cases := []struct{ id, slug, want string }{
		{"ABCDEF12-3456", "site", "zp-abcdef12-site"},
		{"a1b2c3d4e5", "My Site!!", "zp-a1b2c3d4-my-site"},
		{"a1b2c3d4", "", "zp-a1b2c3d4"},
		{"a1b2c3d4", "!!!", "zp-a1b2c3d4"},
	}
	for _, c := range cases {
		if got := ProjectNamespace(c.id, c.slug); got != c.want {
			t.Errorf("ProjectNamespace(%q,%q) = %q, want %q", c.id, c.slug, got, c.want)
		}
	}
	if got := ProjectNamespace("a1b2c3d4", strings.Repeat("x", 100)); len(got) > 63 {
		t.Errorf("namespace too long: %d", len(got))
	}
}
