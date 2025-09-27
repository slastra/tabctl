//go:build windows

package platform

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
)

// RegisterNativeMessagingHost registers a native messaging host in Windows registry
func RegisterNativeMessagingHost(browser, hostName, manifestPath string) error {
	var keyPath string

	switch browser {
	case "firefox":
		keyPath = fmt.Sprintf(`SOFTWARE\Mozilla\NativeMessagingHosts\%s`, hostName)
	case "chrome":
		keyPath = fmt.Sprintf(`SOFTWARE\Google\Chrome\NativeMessagingHosts\%s`, hostName)
	case "chromium":
		keyPath = fmt.Sprintf(`SOFTWARE\Chromium\NativeMessagingHosts\%s`, hostName)
	case "brave":
		keyPath = fmt.Sprintf(`SOFTWARE\BraveSoftware\Brave-Browser\NativeMessagingHosts\%s`, hostName)
	default:
		return fmt.Errorf("unsupported browser: %s", browser)
	}

	// Open or create the registry key
	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create registry key: %w", err)
	}
	defer key.Close()

	// Set the default value to the manifest path
	err = key.SetStringValue("", manifestPath)
	if err != nil {
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	return nil
}

// UnregisterNativeMessagingHost removes a native messaging host from Windows registry
func UnregisterNativeMessagingHost(browser, hostName string) error {
	var keyPath string

	switch browser {
	case "firefox":
		keyPath = fmt.Sprintf(`SOFTWARE\Mozilla\NativeMessagingHosts\%s`, hostName)
	case "chrome":
		keyPath = fmt.Sprintf(`SOFTWARE\Google\Chrome\NativeMessagingHosts\%s`, hostName)
	case "chromium":
		keyPath = fmt.Sprintf(`SOFTWARE\Chromium\NativeMessagingHosts\%s`, hostName)
	case "brave":
		keyPath = fmt.Sprintf(`SOFTWARE\BraveSoftware\Brave-Browser\NativeMessagingHosts\%s`, hostName)
	default:
		return fmt.Errorf("unsupported browser: %s", browser)
	}

	// Delete the registry key
	err := registry.DeleteKey(registry.CURRENT_USER, keyPath)
	if err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to delete registry key: %w", err)
	}

	return nil
}

// MakeWindowsPathDoubleSep converts single backslashes to double backslashes for JSON
func MakeWindowsPathDoubleSep(path string) string {
	// Replace single backslashes with double backslashes for JSON escaping
	result := ""
	for _, char := range path {
		if char == '\\' {
			result += "\\\\"
		} else {
			result += string(char)
		}
	}
	return result
}