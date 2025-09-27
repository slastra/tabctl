package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/platform"
)

// NativeMessagingManifest represents the native messaging host manifest
type NativeMessagingManifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`    // Firefox
	AllowedExtensions []string `json:"allowed_extensions,omitempty"` // Chrome
}

// findMediatorPath finds the tabctl-mediator binary
func findMediatorPath() (string, error) {
	// First, check if tabctl-mediator is in the same directory as tabctl
	execPath, err := os.Executable()
	if err == nil {
		mediatorPath := filepath.Join(filepath.Dir(execPath), "tabctl-mediator")
		if platform.IsWindows() {
			mediatorPath += ".exe"
		}
		if _, err := os.Stat(mediatorPath); err == nil {
			return mediatorPath, nil
		}
	}

	// Check PATH
	mediatorName := "tabctl-mediator"
	if platform.IsWindows() {
		mediatorName += ".exe"
	}

	path, err := exec.LookPath(mediatorName)
	if err == nil {
		return path, nil
	}

	// Try to build it if we're in development
	if _, err := os.Stat("cmd/tabctl-mediator/main.go"); err == nil {
		fmt.Println("Building tabctl-mediator...")
		cmd := exec.Command("go", "build", "-o", mediatorName, "./cmd/tabctl-mediator")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to build mediator: %v", err)
		}

		absPath, _ := filepath.Abs(mediatorName)
		return absPath, nil
	}

	return "", fmt.Errorf("tabctl-mediator not found")
}

// installForBrowser installs the native messaging manifest for a specific browser
func installForBrowser(browser, mediatorPath string, testsMode bool) error {
	// Get manifest directory
	manifestDir, err := platform.GetNativeMessagingHostsDir(browser)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %v", err)
	}

	// Create manifest
	manifest := createManifest(browser, mediatorPath, testsMode)

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	// Write manifest file
	manifestPath := filepath.Join(manifestDir, platform.GetManifestFileName())
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %v", err)
	}

	// Register in Windows registry if needed
	if platform.IsWindows() {
		if err := platform.RegisterNativeMessagingHost(browser, config.NativeHostName, manifestPath); err != nil {
			return fmt.Errorf("failed to register in registry: %v", err)
		}
	}

	return nil
}

// createManifest creates a native messaging manifest for the specified browser
func createManifest(browser, mediatorPath string, testsMode bool) *NativeMessagingManifest {
	// Convert path for Windows JSON escaping
	if platform.IsWindows() {
		mediatorPath = platform.MakeWindowsPathDoubleSep(mediatorPath)
	}

	manifest := &NativeMessagingManifest{
		Name:        config.NativeHostName,
		Description: "TabCtl native messaging host",
		Path:        mediatorPath,
		Type:        "stdio",
	}

	switch browser {
	case "firefox":
		manifest.AllowedExtensions = []string{config.ExtensionID}
	case "chrome", "chromium", "brave":
		// For Chrome, we don't know the extension ID until it's published
		// For development, allow any extension to connect
		origins := []string{
			"chrome-extension://*/",
		}
		if testsMode {
			// Add test extension IDs
			origins = append(origins, "chrome-extension://test-extension-id/")
		}
		manifest.AllowedOrigins = origins
	}

	return manifest
}