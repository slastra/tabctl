package config

import (
	"context"
	"time"
)

// Default timeout values for native messaging
const (
	TransportTimeout = 30 * time.Second

	// MediatorBrowserTimeout bounds how long the mediator waits for the
	// browser extension to answer a command. Kept shorter than the CLI's
	// TransportTimeout so a descriptive error reaches the CLI before its
	// own deadline fires.
	MediatorBrowserTimeout = 25 * time.Second
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

