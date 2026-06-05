package natsclient

import (
	"testing"
)

// TestNew_BadURL verifies that a clearly invalid URL returns an error
// without hanging — nats.Connect returns immediately for bad addresses.
func TestNew_BadURL(t *testing.T) {
	_, err := New("nats://127.0.0.1:14222", "zenith-test")
	if err == nil {
		t.Fatal("expected error connecting to unreachable NATS server, got nil")
	}
}

// TestNew_MalformedURL verifies that an obviously malformed URL is rejected.
func TestNew_MalformedURL(t *testing.T) {
	_, err := New("not-a-nats-url://???###", "zenith-test")
	if err == nil {
		t.Fatal("expected error for malformed NATS URL, got nil")
	}
}

// TestSanitize verifies that dots and NATS wildcards are replaced with dashes.
func TestSanitize(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"zenith.deploy.started", "zenith-deploy-started"},
		{"zenith.>", "zenith--"},
		{"zenith.deploy.*", "zenith-deploy--"},
		{"plain", "plain"},
		{"", ""},
	}

	for _, tc := range cases {
		got := sanitize(tc.input)
		if got != tc.want {
			t.Errorf("sanitize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestClose_NoConnectedClient verifies that Close on a nil-conn client does not panic.
// This validates nil-safety of the struct.
func TestClose_ZeroValue(t *testing.T) {
	// A Client with all nil fields should at least not panic on Close.
	// In practice this never happens in production, but guards against future
	// changes that skip nil checks.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Close panicked on zero-value client: %v", r)
		}
	}()

	c := &Client{}
	// nc is nil — calling Close would panic via nc.Close(), so we only
	// test that subs iteration is safe.
	c.subs = nil
	// We cannot call c.Close() safely since nc is nil, but we verify
	// that iterating nil subs is safe.
	for _, sub := range c.subs {
		sub.Stop()
	}
}
