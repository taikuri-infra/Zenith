package validate

import "testing"

func TestCustomDomain(t *testing.T) {
	const reserved = "zenith.example.com"
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"shop.customer.com", "shop.customer.com", true},
		{"  UPPER.Example.com  ", "upper.example.com", true},
		{"a-b.c1.io", "a-b.c1.io", true},
		{"x`) || PathPrefix(`/", "", false},
		{"zenith.example.com", "", false},
		{"app.zenith.example.com", "", false},
		{"ZENITH.example.com", "", false},
		{"notzenith.example.com", "notzenith.example.com", true},
		{"localhost", "", false},
		{"", "", false},
		{"-bad.com", "", false},
		{"bad-.com", "", false},
		{"a..com", "", false},
		{".a.com", "", false},
		{"a.com.", "", false},
		{"has space.com", "", false},
		{"a/b.com", "", false},
		{"a(b).com", "", false},
		{"under_score.com", "", false},
		{"ünicode.com", "", false},
	}
	long63 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cases = append(cases,
		struct {
			in, want string
			ok       bool
		}{long63 + ".com", long63 + ".com", true},
		struct {
			in, want string
			ok       bool
		}{long63 + "a.com", "", false},
		struct {
			in, want string
			ok       bool
		}{long63 + "." + long63 + "." + long63 + "." + long63 + ".com", "", false}, // > 253
	)
	for _, c := range cases {
		got, err := CustomDomain(c.in, reserved)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("CustomDomain(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("CustomDomain(%q) = %q, want error", c.in, got)
		}
	}
}

func TestCustomDomainNoReserved(t *testing.T) {
	if _, err := CustomDomain("zenith.example.com", ""); err != nil {
		t.Errorf("no reserved domain: unexpected error %v", err)
	}
}

func TestHostname(t *testing.T) {
	if !Hostname("a.example.com") || Hostname("x`) || PathPrefix(`/") || Hostname("A.example.com") {
		t.Error("Hostname mismatch")
	}
}
