package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tabctl/tabctl/internal/browsers"
	"github.com/tabctl/tabctl/internal/config"
)

// NativeMessagingManifest represents the native messaging host manifest
type NativeMessagingManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`    // Chrome/Chromium
	AllowedExtensions []string `json:"allowed_extensions,omitempty"` // Firefox/Zen
}

// installForBrowser writes the native-messaging manifest for one browser.
func installForBrowser(browser browsers.Browser, mediatorPath string) error {
	dir := browser.ManifestDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %v", err)
	}

	data, err := json.MarshalIndent(createManifestForBrowser(browser, mediatorPath), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	manifestPath := filepath.Join(dir, config.NativeHostName+".json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %v", err)
	}

	return nil
}

// createManifestForBrowser creates a native messaging manifest for the specified browser
func createManifestForBrowser(browser browsers.Browser, mediatorPath string) *NativeMessagingManifest {
	manifest := &NativeMessagingManifest{
		Name:        config.NativeHostName,
		Description: "TabCtl native messaging host",
		Path:        mediatorPath,
		Type:        "stdio",
	}

	switch browser.Family {
	case browsers.Firefox:
		manifest.AllowedExtensions = []string{config.ExtensionID}
	case browsers.Chromium:
		// Every Chromium browser loads the same Chrome Web Store build, so
		// they all share one origin.
		manifest.AllowedOrigins = []string{
			fmt.Sprintf("chrome-extension://%s/", config.ChromeID),
		}
	}

	return manifest
}

// findMediatorPath finds the tabctl-mediator binary
func findMediatorPath() (string, error) {
	// First, check if tabctl-mediator is in the same directory as tabctl
	execPath, err := os.Executable()
	if err == nil {
		mediatorPath := filepath.Join(filepath.Dir(execPath), "tabctl-mediator")
		if _, err := os.Stat(mediatorPath); err == nil {
			return mediatorPath, nil
		}
	}

	// Check PATH
	path, err := exec.LookPath("tabctl-mediator")
	if err == nil {
		return path, nil
	}

	// In a source checkout, point at the build that stamps a version rather
	// than building one here. This used to run a bare `go build` into the
	// working directory, which registered an unversioned mediator ("dev" in
	// `tabctl status`) at a second location for the browser to keep
	// launching after the next rebuild.
	if _, err := os.Stat("cmd/tabctl-mediator/main.go"); err == nil {
		return "", fmt.Errorf("tabctl-mediator not found; run `make build`, then `./build/tabctl install`")
	}

	return "", fmt.Errorf("tabctl-mediator not found next to tabctl or on PATH")
}
