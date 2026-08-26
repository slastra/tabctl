package browsers

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tabctl/tabctl/internal/dbus"
)

// A Key becomes a D-Bus bus-name element and an object-path element, so it is
// restricted to [A-Za-z0-9_] and may not lead with a digit. A bad Key does not
// fail at build time; the mediator dies at RequestName on the user's machine.
var busSafe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func TestKeysAreBusSafeUniqueAndSuffixStable(t *testing.T) {
	seenKey := map[string]string{}
	seenDir := map[string]string{}

	for _, b := range All() {
		if !busSafe.MatchString(b.Key) {
			t.Errorf("%s: Key %q is not a valid D-Bus name element", b.Name, b.Key)
		}
		if prev, dup := seenKey[b.Key]; dup {
			t.Errorf("Key %q used by both %q and %q", b.Key, prev, b.Name)
		}
		seenKey[b.Key] = b.Name

		if prev, dup := seenDir[b.ConfigDir]; dup {
			t.Errorf("ConfigDir %q used by both %q and %q", b.ConfigDir, prev, b.Name)
		}
		seenDir[b.ConfigDir] = b.Name

		// Instance naming appends a digit to the Key for a second profile and
		// BaseBrowser strips trailing digits to recover the browser. A Key
		// ending in a digit would make Chrome2 indistinguishable from a
		// browser genuinely named Chrome2.
		if got := dbus.BaseBrowser(b.Key); got != b.Key {
			t.Errorf("%s: Key %q is not stable under BaseBrowser (got %q); it must not end in a digit", b.Name, b.Key, got)
		}
	}
}

// Brave ships two editions across four channels, each with its own profile
// directory. Missing one is exactly the bug this table was built to fix
// (issue #4), and it is invisible until someone with that channel installs.
func TestBraveMatrixIsComplete(t *testing.T) {
	want := map[string]string{
		"Brave-Browser":             "Brave",
		"Brave-Browser-Beta":        "BraveBeta",
		"Brave-Browser-Nightly":     "BraveNightly",
		"Brave-Browser-Development": "BraveDevelopment",
		"Brave-Origin":              "BraveOrigin",
		"Brave-Origin-Beta":         "BraveOriginBeta",
		"Brave-Origin-Nightly":      "BraveOriginNightly",
		"Brave-Origin-Development":  "BraveOriginDevelopment",
	}

	got := map[string]string{}
	for _, b := range All() {
		if strings.HasPrefix(filepath.Base(b.ConfigDir), "Brave-") {
			got[filepath.Base(b.ConfigDir)] = b.Key
		}
	}

	for dir, key := range want {
		if got[dir] != key {
			t.Errorf("profile dir %s: got Key %q, want %q", dir, got[dir], key)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d Brave entries, want %d: %v", len(got), len(want), got)
	}
}

func TestDetectFromExe(t *testing.T) {
	cases := []struct{ exe, want string }{
		// The case from issue #4, and the reason this matches the whole path
		// rather than the basename. Every Brave channel package ships its ELF
		// as "brave"; only the install directory says which channel it is.
		// Verified against a real brave-origin-bin install: the process that
		// spawns the mediator is /opt/brave-origin-bin/brave.
		{"/opt/brave-origin-bin/brave", "BraveOrigin"},
		{"/opt/brave-bin/brave", "Brave"},
		{"/opt/brave-beta-bin/brave", "BraveBeta"},
		{"/opt/brave-nightly-bin/brave", "BraveNightly"},

		// The wrapper scripts, in case a distro execs one of them directly.
		{"/usr/bin/brave-origin", "BraveOrigin"},
		{"/usr/bin/brave-origin-beta", "BraveOriginBeta"},
		{"/usr/bin/brave-origin-nightly", "BraveOriginNightly"},
		{"/usr/bin/brave", "Brave"},
		{"/usr/bin/brave-browser", "Brave"},
		{"/usr/bin/brave-beta", "BraveBeta"},
		{"/usr/bin/brave-browser-nightly", "BraveNightly"},

		// The pre-existing load-bearing ordering, now enforced by length.
		{"/usr/bin/google-chrome-stable", "Chrome"},
		{"/usr/bin/google-chrome", "Chrome"},
		{"/opt/google/chrome/chrome", "Chrome"},
		{"/usr/lib/chromium/chromium", "Chromium"},
		{"/usr/bin/helium", "Helium"},

		{"/usr/bin/lynx", ""},
	}

	for _, c := range cases {
		if got := DetectFromExe(c.exe); got != c.want {
			t.Errorf("DetectFromExe(%q) = %q, want %q", c.exe, got, c.want)
		}
	}
}

// Firefox-family browsers are identified by the manifest path they pass as
// argv[1], which names their profile root.
func TestDetectFromManifestArg(t *testing.T) {
	for _, b := range All() {
		if b.Family != Firefox {
			continue
		}
		arg := filepath.Join(b.ManifestDir(), "dev.slastra.tabctl.json")
		if got := Detect([]string{arg}); got != b.Key {
			t.Errorf("Detect(%q) = %q, want %q", arg, got, b.Key)
		}
	}
}

func TestManifestDirCasing(t *testing.T) {
	for _, b := range All() {
		dir := b.ManifestDir()
		if filepath.Dir(dir) != b.ConfigDir {
			t.Errorf("%s: ManifestDir %q is not under ConfigDir %q", b.Name, dir, b.ConfigDir)
		}
		want := "NativeMessagingHosts"
		if b.Family == Firefox {
			want = "native-messaging-hosts"
		}
		if filepath.Base(dir) != want {
			t.Errorf("%s: ManifestDir base = %q, want %q", b.Name, filepath.Base(dir), want)
		}
	}
}

func TestMatches(t *testing.T) {
	b := Browser{Name: "Brave Origin", Key: "BraveOrigin"}
	for _, target := range []string{"Brave Origin", "brave origin", "BraveOrigin", "braveorigin"} {
		if !b.Matches(target) {
			t.Errorf("Matches(%q) = false, want true", target)
		}
	}
	for _, target := range []string{"Brave", "Origin", "brave-origin", ""} {
		if b.Matches(target) {
			t.Errorf("Matches(%q) = true, want false", target)
		}
	}
}
