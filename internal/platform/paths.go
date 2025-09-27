package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetNativeMessagingHostsDir returns the native messaging hosts directory for the current platform
func GetNativeMessagingHostsDir(browser string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "linux":
		return getLinuxNativeMessagingDir(browser, homeDir), nil
	case "darwin":
		return getDarwinNativeMessagingDir(browser, homeDir), nil
	case "windows":
		return getWindowsNativeMessagingDir(browser)
	default:
		// Fallback to linux paths
		return getLinuxNativeMessagingDir(browser, homeDir), nil
	}
}

// getLinuxNativeMessagingDir returns Linux native messaging directory
func getLinuxNativeMessagingDir(browser, homeDir string) string {
	switch browser {
	case "firefox":
		return filepath.Join(homeDir, ".mozilla", "native-messaging-hosts")
	case "chrome":
		return filepath.Join(homeDir, ".config", "google-chrome", "NativeMessagingHosts")
	case "chromium":
		return filepath.Join(homeDir, ".config", "chromium", "NativeMessagingHosts")
	case "brave":
		return filepath.Join(homeDir, ".config", "BraveSoftware", "Brave-Browser", "NativeMessagingHosts")
	default:
		return filepath.Join(homeDir, ".config", "google-chrome", "NativeMessagingHosts")
	}
}

// getDarwinNativeMessagingDir returns macOS native messaging directory
func getDarwinNativeMessagingDir(browser, homeDir string) string {
	switch browser {
	case "firefox":
		return filepath.Join(homeDir, "Library", "Application Support", "Mozilla", "NativeMessagingHosts")
	case "chrome":
		return filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
	case "chromium":
		return filepath.Join(homeDir, "Library", "Application Support", "Chromium", "NativeMessagingHosts")
	case "brave":
		return filepath.Join(homeDir, "Library", "Application Support", "BraveSoftware", "Brave-Browser", "NativeMessagingHosts")
	default:
		return filepath.Join(homeDir, "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts")
	}
}

// getWindowsNativeMessagingDir returns Windows native messaging directory
func getWindowsNativeMessagingDir(browser string) (string, error) {
	// On Windows, native messaging hosts are typically registered in the registry
	// But we can also use file-based approach
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("USERPROFILE")
		if appData != "" {
			appData = filepath.Join(appData, "AppData", "Roaming")
		}
	}

	switch browser {
	case "firefox":
		return filepath.Join(appData, "Mozilla", "NativeMessagingHosts"), nil
	case "chrome":
		return filepath.Join(appData, "Google", "Chrome", "Application", "NativeMessagingHosts"), nil
	case "chromium":
		return filepath.Join(appData, "Chromium", "Application", "NativeMessagingHosts"), nil
	case "brave":
		return filepath.Join(appData, "BraveSoftware", "Brave-Browser", "Application", "NativeMessagingHosts"), nil
	default:
		return filepath.Join(appData, "Google", "Chrome", "Application", "NativeMessagingHosts"), nil
	}
}

// GetManifestFileName returns the manifest file name for tabctl
func GetManifestFileName() string {
	return "tabctl_mediator.json"
}

// GetManifestPath returns the full path to the manifest file
func GetManifestPath(browser string) (string, error) {
	dir, err := GetNativeMessagingHostsDir(browser)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, GetManifestFileName()), nil
}

// GetConfigDir returns the configuration directory for tabctl
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configDir, "tabctl"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "tabctl"), nil
	default:
		return filepath.Join(homeDir, ".config", "tabctl"), nil
	}
}

// GetCacheDir returns the cache directory for tabctl
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "linux", "darwin":
		cacheDir := os.Getenv("XDG_CACHE_HOME")
		if cacheDir == "" {
			cacheDir = filepath.Join(homeDir, ".cache")
		}
		return filepath.Join(cacheDir, "tabctl"), nil
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		return filepath.Join(localAppData, "tabctl"), nil
	default:
		return filepath.Join(homeDir, ".cache", "tabctl"), nil
	}
}

// GetTempDir returns the temporary directory for tabctl
func GetTempDir() string {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		switch runtime.GOOS {
		case "windows":
			tmpDir = os.Getenv("TEMP")
			if tmpDir == "" {
				tmpDir = os.Getenv("TMP")
			}
			if tmpDir == "" {
				tmpDir = "C:\\temp"
			}
		default:
			tmpDir = "/tmp"
		}
	}
	return filepath.Join(tmpDir, "tabctl")
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetExecutablePath returns the path to the current executable
func GetExecutablePath() (string, error) {
	return os.Executable()
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}