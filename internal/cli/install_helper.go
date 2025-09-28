package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// BrowserInfo contains information about a browser
type BrowserInfo struct {
	Name           string // Display name
	Type           string // "firefox" or "chromium"
	ConfigPath     string // Path to check for browser existence
	NativeHostPath string // Where to install native messaging manifest
}

// getSupportedBrowsers returns the list of browsers we support
func getSupportedBrowsers() []BrowserInfo {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	browsers := []BrowserInfo{
		{
			Name:           "Firefox",
			Type:           "firefox",
			ConfigPath:     filepath.Join(homeDir, ".mozilla"),
			NativeHostPath: filepath.Join(homeDir, ".mozilla", "native-messaging-hosts"),
		},
		{
			Name:           "Zen Browser",
			Type:           "firefox",
			ConfigPath:     filepath.Join(homeDir, ".zen"),
			NativeHostPath: filepath.Join(homeDir, ".zen", "native-messaging-hosts"),
		},
		{
			Name:           "Chrome",
			Type:           "chromium",
			ConfigPath:     filepath.Join(homeDir, ".config", "google-chrome"),
			NativeHostPath: filepath.Join(homeDir, ".config", "google-chrome", "NativeMessagingHosts"),
		},
		{
			Name:           "Chromium",
			Type:           "chromium",
			ConfigPath:     filepath.Join(homeDir, ".config", "chromium"),
			NativeHostPath: filepath.Join(homeDir, ".config", "chromium", "NativeMessagingHosts"),
		},
		{
			Name:           "Brave",
			Type:           "chromium",
			ConfigPath:     filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser"),
			NativeHostPath: filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"),
		},
	}

	// TODO: Add platform-specific paths for Windows and macOS
	return browsers
}

// detectInstalledBrowsers returns a list of browsers that are installed on the system
func detectInstalledBrowsers() []BrowserInfo {
	var detected []BrowserInfo

	for _, browser := range getSupportedBrowsers() {
		if isBrowserInstalled(browser) {
			detected = append(detected, browser)
		}
	}

	return detected
}

// isBrowserInstalled checks if a browser is installed by looking for its config directory
func isBrowserInstalled(browser BrowserInfo) bool {
	if _, err := os.Stat(browser.ConfigPath); err == nil {
		return true
	}

	// Also check if the browser executable exists in PATH
	execName := strings.ToLower(browser.Name)
	if execName == "zen browser" {
		execName = "zen"
	}

	if _, err := exec.LookPath(execName); err == nil {
		return true
	}

	return false
}

// installForBrowserInfo installs the native messaging manifest for a specific browser
func installForBrowserInfo(browser BrowserInfo, mediatorPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(browser.NativeHostPath, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %v", err)
	}

	// Create manifest
	manifest := createManifestForBrowser(browser, mediatorPath)

	// Marshal to JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}

	// Write manifest file
	manifestPath := filepath.Join(browser.NativeHostPath, "tabctl_mediator.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %v", err)
	}

	// Note: Windows registry registration would be handled by the platform package if needed

	return nil
}

// Fixed Chrome extension ID for development builds (generated from manifest key)
const chromeDevExtensionID = "cidpgihmbpgbjanhpkolihgmnhdemdjl"

// createManifestForBrowser creates a native messaging manifest for the specified browser
func createManifestForBrowser(browser BrowserInfo, mediatorPath string) *NativeMessagingManifest {
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

	switch browser.Type {
	case "firefox":
		manifest.AllowedExtensions = []string{config.ExtensionID}
	case "chromium":
		// Use fixed development extension ID for Chrome-based browsers
		manifest.AllowedOrigins = []string{
			fmt.Sprintf("chrome-extension://%s/", chromeDevExtensionID),
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