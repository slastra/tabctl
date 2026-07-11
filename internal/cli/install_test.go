package cli

import (
	"strings"
	"testing"

	"github.com/tabctl/tabctl/internal/config"
)

func TestCreateManifestFirefox(t *testing.T) {
	b := BrowserInfo{Name: "Firefox", Type: "firefox"}
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
	b := BrowserInfo{Name: "Brave", Type: "chromium"}
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
	detected := []BrowserInfo{
		{Name: "Firefox", Type: "firefox"},
		{Name: "Zen Browser", Type: "firefox"},
		{Name: "Brave", Type: "chromium"},
	}

	t.Run("empty target selects all", func(t *testing.T) {
		got, err := filterBrowsersByTarget(detected, "")
		if err != nil || len(got) != 3 {
			t.Fatalf("got %d browsers (err %v), want 3", len(got), err)
		}
	})

	t.Run("exact name match", func(t *testing.T) {
		got, err := filterBrowsersByTarget(detected, "brave")
		if err != nil || len(got) != 1 || got[0].Name != "Brave" {
			t.Fatalf("got %v (err %v), want [Brave]", got, err)
		}
	})

	t.Run("first-word match", func(t *testing.T) {
		got, err := filterBrowsersByTarget(detected, "Zen")
		if err != nil || len(got) != 1 || got[0].Name != "Zen Browser" {
			t.Fatalf("got %v (err %v), want [Zen Browser]", got, err)
		}
	})

	t.Run("no match errors", func(t *testing.T) {
		_, err := filterBrowsersByTarget(detected, "Safari")
		if err == nil || !strings.Contains(err.Error(), "Firefox") {
			t.Errorf("want error listing detected browsers, got %v", err)
		}
	})
}
