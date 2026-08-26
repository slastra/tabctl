package browsers

import (
	"os"
	"path/filepath"
)

// All returns every supported browser. Returns nil if the home directory
// cannot be determined, since every entry is rooted there.
func All() []Browser {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	config := filepath.Join(home, ".config")

	list := []Browser{
		{
			Name:      "Firefox",
			Key:       "Firefox",
			Family:    Firefox,
			ExecNames: []string{"firefox", "firefox-esr"},
			ConfigDir: filepath.Join(home, ".mozilla"),
		},
		{
			Name:      "Zen Browser",
			Key:       "Zen",
			Family:    Firefox,
			ExecNames: []string{"zen", "zen-browser"},
			ConfigDir: filepath.Join(home, ".zen"),
		},
		{
			Name:      "Chrome",
			Key:       "Chrome",
			Family:    Chromium,
			ExecNames: []string{"google-chrome-stable", "google-chrome", "chrome"},
			ConfigDir: filepath.Join(config, "google-chrome"),
		},
		{
			Name:      "Chromium",
			Key:       "Chromium",
			Family:    Chromium,
			ExecNames: []string{"chromium-browser", "chromium"},
			ConfigDir: filepath.Join(config, "chromium"),
		},
		{
			Name:      "Helium",
			Key:       "Helium",
			Family:    Chromium,
			ExecNames: []string{"helium"},
			ConfigDir: filepath.Join(config, "net.imput.helium"),
		},
	}
	return append(list, braveChannels(config)...)
}

// braveChannels generates Brave's release matrix: two editions across four
// channels.
//
// Brave ships Origin as a separate compile-out build, not a flag on Brave, so
// the two can be installed side by side with independent profiles. Both
// editions carry the full channel set, which is eight directories under
// ~/.config/BraveSoftware. Generating them keeps a third edition to a two-line
// change, and keeps the Origin rows from drifting away from the Browser rows.
//
// The directory names are not guessed: 1Password's native-messaging installer
// targets this exact matrix, and all eight appear on a machine with it
// installed.
func braveChannels(config string) []Browser {
	// dirEdition is Brave's profile-directory spelling; execEdition is the
	// binary's. They differ: ~/.config/BraveSoftware/Brave-Browser is driven
	// by /usr/bin/brave.
	editions := []struct{ dirEdition, execEdition, label string }{
		{"Browser", "brave", ""},
		{"Origin", "brave-origin", "Origin"},
	}
	// Stable has no directory suffix and no binary suffix. The rest append
	// both, lowercased for the binary. "Development" has no packaged binary we
	// know of, so it also probes the conventional "-dev" spelling; its profile
	// directory is what actually detects it.
	channels := []struct{ dirSuffix, execSuffix, label string }{
		{"", "", ""},
		{"-Beta", "-beta", "Beta"},
		{"-Nightly", "-nightly", "Nightly"},
		{"-Development", "-development", "Development"},
	}

	var out []Browser
	for _, e := range editions {
		for _, c := range channels {
			name, key := "Brave", "Brave"
			for _, part := range []string{e.label, c.label} {
				if part != "" {
					name += " " + part
					key += part
				}
			}

			execs := []string{e.execEdition + c.execSuffix}
			if c.label == "Development" {
				execs = append(execs, e.execEdition+"-dev")
			}
			if e.dirEdition == "Browser" {
				// Brave's own .deb installs brave-browser*; the Arch package
				// installs brave*. Probe both.
				execs = append(execs, "brave-browser"+c.execSuffix)
			}

			out = append(out, Browser{
				Name:      name,
				Key:       key,
				Family:    Chromium,
				ExecNames: execs,
				ConfigDir: filepath.Join(config, "BraveSoftware", "Brave-"+e.dirEdition+c.dirSuffix),
			})
		}
	}
	return out
}

// Installed returns the subset of All that appears to be present.
func Installed() []Browser {
	var found []Browser
	for _, b := range All() {
		if b.Installed() {
			found = append(found, b)
		}
	}
	return found
}
