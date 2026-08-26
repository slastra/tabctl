package browsers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Detect identifies the browser that launched this process, given the
// non-flag command-line arguments.
//
// Firefox-family browsers pass the manifest path they loaded us from as
// argv[1], which names the profile root directly. Chromium-family browsers
// pass only the chrome-extension:// origin, which is identical across every
// Chromium browser, so they are identified by the parent process binary.
//
// Returns "Unknown" when neither route resolves; the mediator still runs and
// claims that name, so an unrecognised browser degrades to a working but
// oddly-labelled instance rather than a failure.
func Detect(args []string) string {
	if key := detectFromManifestArg(args); key != "" {
		return key
	}
	if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", os.Getppid())); err == nil {
		if key := DetectFromExe(exe); key != "" {
			return key
		}
	}
	return "Unknown"
}

// detectFromManifestArg matches the manifest path Firefox-family browsers pass
// against each browser's profile directory name, so ~/.zen/... resolves to Zen.
func detectFromManifestArg(args []string) string {
	for _, arg := range args {
		for _, b := range All() {
			if b.Family != Firefox {
				continue
			}
			if strings.Contains(arg, string(filepath.Separator)+filepath.Base(b.ConfigDir)+string(filepath.Separator)) {
				return b.Key
			}
		}
	}
	return ""
}

// DetectFromExe maps a browser executable path to a browser Key, matching the
// longest ExecName anywhere in the path.
//
// Matching the whole path rather than the basename is load-bearing, not
// belt-and-braces. Brave Origin's package ships /usr/bin/brave-origin, a script
// that execs /opt/brave-origin-bin/brave-origin, which is a second script that
// execs /opt/brave-origin-bin/brave. That last one is the process that spawns
// us, and its basename is plain "brave" -- identical to Brave's. The install
// directory is the only thing in the path that distinguishes the two. The same
// holds across the channel packages, which all ship their ELF as "brave" under
// /opt/brave-<channel>-bin.
//
// Longest-match is what makes the table self-maintaining. The old hand-ordered
// switch was correct only because "google-chrome" happened to be tested before
// "chrome", with nothing recording that the order was load-bearing.
func DetectFromExe(exe string) string {
	path := strings.ToLower(exe)

	type probe struct{ exec, key string }
	var probes []probe
	for _, b := range All() {
		for _, name := range b.ExecNames {
			probes = append(probes, probe{name, b.Key})
		}
	}
	sort.Slice(probes, func(i, j int) bool {
		if len(probes[i].exec) != len(probes[j].exec) {
			return len(probes[i].exec) > len(probes[j].exec)
		}
		return probes[i].exec < probes[j].exec
	})

	for _, p := range probes {
		if strings.Contains(path, p.exec) {
			return p.key
		}
	}
	return ""
}
