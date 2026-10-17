package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tabctl/tabctl/internal/browsers"
	"github.com/tabctl/tabctl/internal/config"
)

func TestCreateManifestFirefox(t *testing.T) {
	b := browsers.Browser{Name: "Firefox", Family: browsers.Firefox}
	m := createManifestForBrowser(b, "/usr/bin/tabctl-mediator")

	if m.Name != config.NativeHostName || m.Path != "/usr/bin/tabctl-mediator" || m.Type != "stdio" {
		t.Fatalf("unexpected manifest core: %+v", m)
	}
	if len(m.AllowedExtensions) != 1 || m.AllowedExtensions[0] != config.ExtensionID {
		t.Errorf("AllowedExtensions: got %v, want [%s]", m.AllowedExtensions, config.ExtensionID)
	}
	if m.AllowedOrigins != nil {
		t.Errorf("Firefox manifest should not set AllowedOrigins, got %v", m.AllowedOrigins)
	}
}

func TestCreateManifestChromium(t *testing.T) {
	b := browsers.Browser{Name: "Brave", Family: browsers.Chromium}
	m := createManifestForBrowser(b, "/usr/bin/tabctl-mediator")

	wantOrigin := "chrome-extension://" + config.ChromeID + "/"
	if len(m.AllowedOrigins) != 1 || m.AllowedOrigins[0] != wantOrigin {
		t.Errorf("AllowedOrigins: got %v, want [%s]", m.AllowedOrigins, wantOrigin)
	}
	if m.AllowedExtensions != nil {
		t.Errorf("Chromium manifest should not set AllowedExtensions, got %v", m.AllowedExtensions)
	}
}

func TestFilterBrowsersByTarget(t *testing.T) {
	detected := []browsers.Browser{
		{Name: "Firefox", Key: "Firefox", Family: browsers.Firefox},
		{Name: "Zen Browser", Key: "Zen", Family: browsers.Firefox},
		{Name: "Brave", Key: "Brave", Family: browsers.Chromium},
		{Name: "Brave Origin", Key: "BraveOrigin", Family: browsers.Chromium},
		{Name: "Brave Origin Beta", Key: "BraveOriginBeta", Family: browsers.Chromium},
	}

	t.Run("empty target selects all", func(t *testing.T) {
		got, err := filterBrowsersByTarget(detected, "")
		if err != nil || len(got) != len(detected) {
			t.Fatalf("got %d browsers (err %v), want %d", len(got), err, len(detected))
		}
	})

	t.Run("exact name match", func(t *testing.T) {
		got, err := filterBrowsersByTarget(detected, "brave")
		if err != nil || len(got) != 1 || got[0].Name != "Brave" {
			t.Fatalf("got %v (err %v), want [Brave]", browserNames(got), err)
		}
	})

	// The old first-word matcher would have returned all three Brave rows
	// here, silently installing for browsers the user did not name.
	t.Run("brave does not sweep in Origin", func(t *testing.T) {
		got, _ := filterBrowsersByTarget(detected, "Brave")
		if len(got) != 1 {
			t.Fatalf("--browser brave selected %v, want only Brave", browserNames(got))
		}
	})

	t.Run("key match", func(t *testing.T) {
		got, err := filterBrowsersByTarget(detected, "Zen")
		if err != nil || len(got) != 1 || got[0].Name != "Zen Browser" {
			t.Fatalf("got %v (err %v), want [Zen Browser]", browserNames(got), err)
		}
	})

	t.Run("multi-word name, with and without the space", func(t *testing.T) {
		for _, target := range []string{"Brave Origin", "braveorigin", "BRAVEORIGIN"} {
			got, err := filterBrowsersByTarget(detected, target)
			if err != nil || len(got) != 1 || got[0].Key != "BraveOrigin" {
				t.Errorf("target %q: got %v (err %v), want [Brave Origin]", target, browserNames(got), err)
			}
		}
	})

	t.Run("no match errors", func(t *testing.T) {
		_, err := filterBrowsersByTarget(detected, "Safari")
		if err == nil || !strings.Contains(err.Error(), "Firefox") {
			t.Errorf("want error listing detected browsers, got %v", err)
		}
	})
}

func TestUninstallRemovesOnlyOurManifest(t *testing.T) {
	root := t.TempDir()
	withManifest := browsers.Browser{Name: "Brave Beta", Family: browsers.Chromium, ConfigDir: filepath.Join(root, "brave-beta")}
	without := browsers.Browser{Name: "Zen Browser", Family: browsers.Firefox, ConfigDir: filepath.Join(root, "zen")}

	if err := installForBrowser(withManifest, "/usr/bin/tabctl-mediator"); err != nil {
		t.Fatal(err)
	}
	// Another host's manifest in the same directory must survive.
	other := filepath.Join(withManifest.ManifestDir(), "com.1password.1password.json")
	if err := os.WriteFile(other, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := uninstallFrom([]browsers.Browser{withManifest, without})
	if err != nil {
		t.Fatalf("uninstallFrom: %v", err)
	}
	want := filepath.Join(withManifest.ManifestDir(), config.NativeHostName+".json")
	if len(removed) != 1 || removed[0] != want {
		t.Errorf("removed = %v, want [%s]", removed, want)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("manifest still present: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated manifest was touched: %v", err)
	}

	// Running it again is a no-op, not an error.
	removed, err = uninstallFrom([]browsers.Browser{withManifest, without})
	if err != nil || len(removed) != 0 {
		t.Errorf("second run: removed %v, err %v; want nothing, nil", removed, err)
	}
}
