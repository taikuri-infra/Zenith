package upgrade

import "testing"

func TestValidateVersionPrerelease(t *testing.T) {
	for _, ok := range []string{"1.5.0", "0.1.0-alpha.3", "0.1.0-alpha.3-test", "0.1.0-rc-1.2"} {
		if err := validateVersion(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"v1.5.0", "1.5", "1.5.0-", "1.5.0 ", "latest"} {
		if err := validateVersion(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
