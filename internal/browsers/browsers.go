// Package browsers is the single source of truth for the browsers tabctl
// supports.
//
// Two consumers used to keep separate hand-written lists and drift apart:
// `tabctl install` decided where to write native-messaging manifests, and the
// mediator decided what to call itself on the bus. Nothing made them agree, so
// they didn't (install called Zen "Zen Browser"; the mediator called it "Zen").
// Adding a browser now means adding one row here.
package browsers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Family is a browser's native-messaging dialect. It selects the manifest key
// that lists permitted callers ("allowed_extensions" vs "allowed_origins") and
// the case of the manifest directory.
type Family string

const (
	Firefox  Family = "firefox"
	Chromium Family = "chromium"
)

// Browser describes one installable browser.
//
// Name and Key are deliberately separate. Name is for humans and may contain
// spaces; Key is the identity token, and flows into a D-Bus name element
// (dbus.ServiceName), an object path (dbus.ObjectPath), and the lowercased
// user-facing tab-ID prefix ("braveorigin.1.42"). Deriving Key from Name is
// what produced the old `if execName == "zen browser"` patch, so it is stated
// rather than computed. TestKeysAreBusSafe enforces the constraint.
type Browser struct {
	Name      string
	Key       string
	Family    Family
	ExecNames []string // PATH probes; order here is irrelevant, see Detect
	ConfigDir string   // absolute profile root, e.g. ~/.config/BraveSoftware/Brave-Origin
}

// ManifestDir is where this browser looks for native-messaging manifests.
//
// Always derived, never stored. Every row used to spell its config path twice,
// once as ConfigPath and again as the parent of NativeHostPath, which is six
// chances to typo one and not notice until a user reports the browser "isn't
// detected".
func (b Browser) ManifestDir() string {
	if b.Family == Firefox {
		return filepath.Join(b.ConfigDir, "native-messaging-hosts")
	}
	return filepath.Join(b.ConfigDir, "NativeMessagingHosts")
}

// Installed reports whether the browser appears to be present: either it has
// created its profile directory, or one of its binaries is on PATH. The PATH
// probe covers a browser installed but never launched, which has no profile
// yet.
func (b Browser) Installed() bool {
	if _, err := os.Stat(b.ConfigDir); err == nil {
		return true
	}
	for _, name := range b.ExecNames {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}

// Matches reports whether a --browser value selects this browser. Name, Key,
// and Name with spaces removed all work, so "Brave Origin", "BraveOrigin", and
// "braveorigin" are equivalent.
//
// Matching is exact. It used to accept the first word of Name, which was
// harmless when "Zen Browser" was the only multi-word entry but silently
// selects all eight Brave channels once they exist.
func (b Browser) Matches(target string) bool {
	return strings.EqualFold(b.Name, target) ||
		strings.EqualFold(b.Key, target) ||
		strings.EqualFold(strings.ReplaceAll(b.Name, " ", ""), target)
}
