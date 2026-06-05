package semver

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input     string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{"1.2.3", 1, 2, 3, false},
		{"v1.2.3", 1, 2, 3, false},
		{"zenith-1.2.3", 1, 2, 3, false},
		{"0.9.0", 0, 9, 0, false},
		{"latest", 0, 0, 0, true},
		{"", 0, 0, 0, true},
		{"1.2", 0, 0, 0, true},
		{"abc", 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if major != tt.wantMajor || minor != tt.wantMinor || patch != tt.wantPatch {
				t.Errorf("ParseVersion(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, major, minor, patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
		})
	}
}

func TestIsSafeUpgrade(t *testing.T) {
	tests := []struct {
		current  string
		target   string
		wantSafe bool
	}{
		{"1.0.0", "1.1.0", true},  // one minor version up: OK
		{"1.0.0", "1.0.5", true},  // patch only: OK
		{"1.0.0", "1.2.0", false}, // two minor versions: blocked
		{"1.0.0", "1.5.0", false}, // five minor versions: blocked
		{"1.1.0", "2.0.0", true},  // major bump, one step: OK
		{"0.9.0", "1.0.0", true},  // cross-major, one step: OK
		{"0.9.0", "1.1.0", false}, // cross-major + minor: blocked
		{"1.2.3", "1.3.0", true},  // minor +1: OK
		{"1.2.3", "1.2.10", true}, // patch only: OK
		{"1.1.0", "1.0.0", false}, // minor downgrade: blocked
		{"2.0.0", "1.9.0", false}, // major downgrade: blocked
		{"1.5.0", "1.4.9", false}, // patch downgrade: blocked
	}
	for _, tt := range tests {
		t.Run(tt.current+"→"+tt.target, func(t *testing.T) {
			err := IsSafeUpgrade(tt.current, tt.target)
			isSafe := (err == nil)
			if isSafe != tt.wantSafe {
				t.Errorf("IsSafeUpgrade(%q, %q) safe=%v, want %v (err: %v)",
					tt.current, tt.target, isSafe, tt.wantSafe, err)
			}
		})
	}
}
