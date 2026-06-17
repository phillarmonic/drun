package app

import (
	"strings"
	"testing"
)

func TestReleaseBinaryName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		goos    string
		goarch  string
		want    string
		wantErr string
	}{
		{name: "darwin arm64", goos: "darwin", goarch: "arm64", want: "xdrun-darwin-arm64"},
		{name: "windows amd64", goos: "windows", goarch: "amd64", want: "xdrun-windows-amd64.exe"},
		{name: "unsupported arch", goos: "linux", goarch: "386", wantErr: "unsupported architecture"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := releaseBinaryName(tt.goos, tt.goarch)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("releaseBinaryName() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("releaseBinaryName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindAssetDownloadURL(t *testing.T) {
	t.Parallel()

	release := GitHubRelease{
		TagName: "v2.19.0",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: "xdrun-darwin-arm64", BrowserDownloadURL: "https://example.com/xdrun-darwin-arm64"},
			{Name: "xdrun-linux-amd64", BrowserDownloadURL: "https://example.com/xdrun-linux-amd64"},
		},
	}

	got, err := findAssetDownloadURL(release, "xdrun-darwin-arm64")
	if err != nil {
		t.Fatalf("findAssetDownloadURL() error = %v", err)
	}
	if got != "https://example.com/xdrun-darwin-arm64" {
		t.Fatalf("findAssetDownloadURL() = %q", got)
	}

	_, err = findAssetDownloadURL(release, "xdrun-windows-arm64.exe")
	if err == nil {
		t.Fatal("expected missing asset error")
	}
	if !strings.Contains(err.Error(), "available: xdrun-darwin-arm64, xdrun-linux-amd64") {
		t.Fatalf("unexpected error: %v", err)
	}
}
