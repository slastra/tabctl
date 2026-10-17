package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// extensionsDir is the extension source, relative to this package.
var extensionsDir = filepath.Join("..", "..", "extensions")

// readExtensionFile returns a file under extensions/, skipping the test when
// it is absent: a packager may build from a tree without the extension
// sources, and that is no reason to fail their build.
func readExtensionFile(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(extensionsDir, rel))
	if os.IsNotExist(err) {
		t.Skipf("%s not present in this tree", rel)
	}
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// The protocol version lives twice: here, and as PROTOCOL_VERSION in the
// extension's core.js. They ship through different channels (AUR and the
// stores), so a release that bumps one and not the other would show every
// user an out-of-date badge and a protocol-mismatch error.
func TestProtocolVersionMatchesExtension(t *testing.T) {
	src := readExtensionFile(t, filepath.Join("shared", "core.js"))
	m := regexp.MustCompile(`(?m)^const PROTOCOL_VERSION = (\d+);`).FindSubmatch(src)
	if m == nil {
		t.Fatal("PROTOCOL_VERSION declaration not found in core.js")
	}
	got, _ := strconv.Atoi(string(m[1]))
	if got != ProtocolVersion {
		t.Errorf("core.js PROTOCOL_VERSION = %d, config.ProtocolVersion = %d", got, ProtocolVersion)
	}
}

// Both store builds come from one shared source and are released together,
// so their manifest versions must agree.
func TestExtensionManifestVersionsAgree(t *testing.T) {
	version := func(browser string) string {
		var m struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(readExtensionFile(t, filepath.Join(browser, "manifest.json")), &m); err != nil {
			t.Fatalf("%s manifest: %v", browser, err)
		}
		return m.Version
	}
	if c, f := version("chrome"), version("firefox"); c != f {
		t.Errorf("chrome manifest is %s, firefox manifest is %s", c, f)
	}
}
