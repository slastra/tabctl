package config

import (
	"time"
)

// MediatorConfig holds configuration for the mediator
type MediatorConfig struct {
	Host         string
	Port         int
	Debug        bool
	PollInterval time.Duration
	Timeout      time.Duration
	StdioMode    bool // True when mediator runs in stdio mode for native messaging
}

// DefaultMediatorConfig returns the default mediator configuration
func DefaultMediatorConfig() *MediatorConfig {
	return &MediatorConfig{
		Host:         "localhost",
		Port:         4625,
		Debug:        false,
		PollInterval: 100 * time.Millisecond,
		Timeout:      10 * time.Second,
	}
}

// Default regex patterns for text/HTML extraction
const (
	DefaultGetWordsMatchRegex     = `\w+`
	DefaultGetWordsJoinWith       = "\n"
	DefaultGetTextDelimiterRegex  = `\n|\r|\t`
	DefaultGetTextReplaceWith     = " "
	DefaultGetHTMLDelimiterRegex  = `\n|\r|\t`
	DefaultGetHTMLReplaceWith     = " "
)

// Default ports for mediators
var DefaultMediatorPorts = []int{4625, 4626, 4627}

// Default timeout values
const (
	StdinTimeout     = 100 * time.Millisecond
	ShutdownTimeout  = 5 * time.Second
	TransportTimeout = 30 * time.Second
)

// Native messaging host names
const (
	NativeHostName = "tabctl_mediator"
	ExtensionID    = "tabctl@slastra.github.io" // Firefox
	ChromeID       = ""  // Chrome/Chromium - will be generated when published
)

// Browser names
const (
	BrowserFirefox  = "firefox"
	BrowserChrome   = "chrome"
	BrowserChromium = "chromium"
	BrowserBrave    = "brave"
)

// Environment variables
const (
	EnvDebug    = "TABCTL_DEBUG"
	EnvLogFile  = "TABCTL_LOG"
	EnvPort     = "TABCTL_PORT"
	EnvHost     = "TABCTL_HOST"
)

// GetBrowserName returns the browser name from environment or defaults
func GetBrowserName() string {
	// TODO: Detect browser from parent process or environment
	return "unknown"
}

// IsDebugMode checks if debug mode is enabled
func IsDebugMode() bool {
	// TODO: Check environment variable
	return false
}