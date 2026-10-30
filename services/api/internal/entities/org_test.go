package entities

import "testing"

func TestOrgSlug(t *testing.T) {
	cases := map[string]string{
		"Backend":                            "backend",
		"  My Team!! ":                       "my-team",
		"a/b_c":                              "a-b-c",
		"!!!":                                "",
		"Ünï":                                "n",
		"0123456789012345678901234567890123": "012345678901234567890123456789",
	}
	for in, want := range cases {
		if got := OrgSlug(in); got != want {
			t.Errorf("OrgSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOrgNamespace(t *testing.T) {
	got := OrgNamespace("proj-ABCDEF12-3456", "backend")
	if got != "zo-abcdef12-backend" {
		t.Fatalf("got %q", got)
	}
	if len(OrgNamespace("proj-x", OrgSlug("0123456789012345678901234567890123"))) > 63 {
		t.Fatal("namespace exceeds 63 chars")
	}
}
