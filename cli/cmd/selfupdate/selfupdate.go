package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dotechhq/zenith/cli/cmd/version"
	"github.com/spf13/cobra"
)

const releaseAPI = "https://api.github.com/repos/taikuri-infra/Zenith/releases/latest"

// Cmd is `zen self-update` — replaces the running binary with the latest release.
var Cmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update the zen CLI to the latest release",
	Long:  "Download the latest zen release for this OS/arch, verify its checksum, and replace this binary.",
	RunE:  runSelfUpdate,
}

// assetName is the release asset for the current platform, e.g. zen_linux_amd64.
func assetName() string {
	n := fmt.Sprintf("zen_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		n += ".exe"
	}
	return n
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease(apiURL string) (*ghRelease, error) {
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func assetURL(rel *ghRelease, name string) string {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a.URL
		}
	}
	return ""
}

// checksumFor parses a `<sha256>  <name>` checksums file and returns the hash for name.
func checksumFor(checksums, name string) string {
	for _, line := range strings.Split(checksums, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[1] == name {
			return f[0]
		}
	}
	return ""
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	rel, err := latestRelease(releaseAPI)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}
	if rel.TagName == version.Version {
		fmt.Printf("Already on the latest version (%s).\n", version.Version)
		return nil
	}
	name := assetName()
	binURL := assetURL(rel, name)
	if binURL == "" {
		return fmt.Errorf("release %s has no asset %q for this platform", rel.TagName, name)
	}

	fmt.Printf("Updating zen %s -> %s ...\n", version.Version, rel.TagName)

	data, err := download(binURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", name, err)
	}
	// Verify checksum when the release ships one.
	if sumURL := assetURL(rel, "zen_checksums.txt"); sumURL != "" {
		sums, derr := download(sumURL)
		if derr != nil {
			return fmt.Errorf("download checksums: %w", derr)
		}
		want := checksumFor(string(sums), name)
		got := fmt.Sprintf("%x", sha256.Sum256(data))
		if want == "" || !equalHex(want, got) {
			return fmt.Errorf("checksum mismatch for %s (refusing to install)", name)
		}
	}
	if err := replaceSelf(data); err != nil {
		return err
	}
	fmt.Printf("Updated to %s. Run `zen version` to confirm.\n", rel.TagName)
	return nil
}

func download(url string) ([]byte, error) {
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func equalHex(a, b string) bool {
	da, err1 := hex.DecodeString(strings.TrimSpace(a))
	db, err2 := hex.DecodeString(strings.TrimSpace(b))
	if err1 != nil || err2 != nil || len(da) != len(db) {
		return false
	}
	var diff byte
	for i := range da {
		diff |= da[i] ^ db[i]
	}
	return diff == 0
}

// replaceSelf atomically swaps the running binary for new content.
func replaceSelf(data []byte) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, rerr := filepath.EvalSymlinks(self); rerr == nil {
		self = resolved
	}
	dir := filepath.Dir(self)
	tmp, err := os.CreateTemp(dir, ".zen-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (try: sudo zen self-update): %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpName, self); err != nil {
		return fmt.Errorf("replace binary at %s (try: sudo zen self-update): %w", self, err)
	}
	return nil
}
