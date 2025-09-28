//go:build !windows

package platform

// RegisterNativeMessagingHost is a no-op on Unix systems (uses file-based registration)
func RegisterNativeMessagingHost(browser, hostName, manifestPath string) error {
	// On Unix systems, native messaging hosts are registered by placing
	// manifest files in the appropriate directories, not via registry
	return nil
}

// UnregisterNativeMessagingHost is a no-op on Unix systems
func UnregisterNativeMessagingHost(browser, hostName string) error {
	// On Unix systems, unregistration is done by removing the manifest file
	return nil
}

// MakeWindowsPathDoubleSep is a no-op on Unix systems
func MakeWindowsPathDoubleSep(path string) string {
	// No need to escape backslashes on Unix systems
	return path
}