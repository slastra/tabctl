package config

import (
	"context"
	"time"
)

// Default timeout values for native messaging
const (
	TransportTimeout = 30 * time.Second
)

// CommandContext returns a context with the default command timeout.
func CommandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TransportTimeout)
}

// Native messaging host names
const (
	NativeHostName = "tabctl_mediator"
	ExtensionID    = "tabctl@slastra.github.io"   // Firefox
	ChromeID       = "baomblllgemcgbignhpbipgiofmjdhpn" // Chrome/Chromium/Brave
)

// Browser names
const (
	BrowserFirefox  = "firefox"
	BrowserChrome   = "chrome"
	BrowserChromium = "chromium"
	BrowserBrave    = "brave"
)