package client

import "testing"

func TestComposeAndParseRoundTrip(t *testing.T) {
	c := &DBusClient{browser: "Firefox", prefix: determinePrefixForBrowser("Firefox")}

	id := c.composeID(1, 123)
	if id != "firefox.1.123" {
		t.Fatalf("composeID: got %q, want firefox.1.123", id)
	}

	if got := routePrefix(id); got != "firefox." {
		t.Errorf("routePrefix: got %q, want firefox.", got)
	}
	n, err := numericTabID(id)
	if err != nil || n != 123 {
		t.Errorf("numericTabID: got %d (err %v), want 123", n, err)
	}
}

func TestNumericTabIDInvalid(t *testing.T) {
	if _, err := numericTabID("firefox.1.notanumber"); err == nil {
		t.Error("expected error for non-numeric tab ID")
	}
}

func TestPrefixLowercasesBrowser(t *testing.T) {
	if got := determinePrefixForBrowser("Brave"); got != "brave." {
		t.Errorf("got %q, want brave.", got)
	}
}
