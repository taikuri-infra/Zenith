package selfupdate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestAssetName(t *testing.T) {
	want := fmt.Sprintf("zen_%s_%s", runtime.GOOS, runtime.GOARCH)
	if got := assetName(); got != want && got != want+".exe" {
		t.Errorf("assetName() = %q, want %q", got, want)
	}
}

func TestAssetURL(t *testing.T) {
	rel := &ghRelease{}
	rel.Assets = append(rel.Assets, struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	}{Name: "zen_linux_amd64", URL: "https://x/zen_linux_amd64"})
	if url := assetURL(rel, "zen_linux_amd64"); url != "https://x/zen_linux_amd64" {
		t.Errorf("assetURL = %q", url)
	}
	if url := assetURL(rel, "zen_darwin_arm64"); url != "" {
		t.Errorf("missing asset should be empty, got %q", url)
	}
}

func TestChecksumFor(t *testing.T) {
	sums := "abc123  zen_linux_amd64\ndef456  zen_darwin_arm64\n"
	if h := checksumFor(sums, "zen_darwin_arm64"); h != "def456" {
		t.Errorf("checksumFor = %q, want def456", h)
	}
	if h := checksumFor(sums, "zen_windows_amd64.exe"); h != "" {
		t.Errorf("missing checksum should be empty, got %q", h)
	}
}

func TestEqualHex(t *testing.T) {
	if !equalHex("aabb", "aabb") {
		t.Error("equal hex should match")
	}
	if equalHex("aabb", "aabbcc") {
		t.Error("different lengths must not match")
	}
	if equalHex("aabb", "aabc") {
		t.Error("different values must not match")
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"zen-v0.2.0","assets":[{"name":"zen_linux_amd64","browser_download_url":"https://x/z"}]}`))
	}))
	defer srv.Close()

	rel, err := latestRelease(srv.URL)
	if err != nil {
		t.Fatalf("latestRelease: %v", err)
	}
	if rel.TagName != "zen-v0.2.0" {
		t.Errorf("tag = %q", rel.TagName)
	}
	if assetURL(rel, "zen_linux_amd64") != "https://x/z" {
		t.Error("asset URL not parsed")
	}
}
