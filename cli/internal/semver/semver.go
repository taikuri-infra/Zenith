package semver

import (
	"fmt"
	"regexp"
	"strconv"
)

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// ParseVersion extracts major, minor, patch from strings like "1.2.3", "v1.2.3", "zenith-1.2.3".
func ParseVersion(v string) (major, minor, patch int, err error) {
	m := versionRe.FindStringSubmatch(v)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("cannot parse version from %q", v)
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	return major, minor, patch, nil
}

// IsSafeUpgrade returns nil if current→target is a safe upgrade (≤1 minor version jump).
func IsSafeUpgrade(current, target string) error {
	curMaj, curMin, _, err := ParseVersion(current)
	if err != nil {
		return fmt.Errorf("current version: %w", err)
	}
	tgtMaj, tgtMin, _, err := ParseVersion(target)
	if err != nil {
		return fmt.Errorf("target version: %w", err)
	}

	if tgtMaj < curMaj || (tgtMaj == curMaj && tgtMin < curMin) {
		return fmt.Errorf(
			"downgrade from v%d.%d to v%d.%d is not supported; restore from backup if needed",
			curMaj, curMin, tgtMaj, tgtMin,
		)
	}

	if curMaj == tgtMaj {
		// Same major: only one minor version increment allowed
		if tgtMin-curMin > 1 {
			return fmt.Errorf(
				"cannot upgrade from v%d.%d to v%d.%d: must upgrade one minor version at a time (next: v%d.%d)",
				curMaj, curMin, tgtMaj, tgtMin, curMaj, curMin+1,
			)
		}
	} else {
		// Cross-major: only one major version step, and must land at minor 0
		if tgtMaj-curMaj > 1 {
			return fmt.Errorf(
				"cannot upgrade from v%d to v%d: must upgrade one major version at a time",
				curMaj, tgtMaj,
			)
		}
		if tgtMin > 0 {
			return fmt.Errorf(
				"cannot upgrade from v%d.%d to v%d.%d: when crossing major versions, must target v%d.0 first",
				curMaj, curMin, tgtMaj, tgtMin, tgtMaj,
			)
		}
	}
	return nil
}
