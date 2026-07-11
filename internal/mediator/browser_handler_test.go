package mediator

import "testing"

// TestHandlerListTabsMapping verifies structured TabData maps to dbus.TabInfo
// with correct numeric conversions.
func TestHandlerListTabsMapping(t *testing.T) {
	want := []TabData{
		{WindowID: 3, TabID: 42, Title: "T", URL: "u", Index: 1, Active: true, Pinned: false},
	}
	mock := newMockTransport(echoResult(want))
	handler := NewDBusHandler(NewBrowserAPI(mock))

	infos, err := handler.ListTabs()
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("got %d tabs, want 1", len(infos))
	}
	got := infos[0]
	if got.WindowID != 3 || got.TabID != 42 || got.Index != 1 || !got.Active || got.Pinned {
		t.Errorf("unexpected mapping: %+v", got)
	}
}

// TestHandlerCloseTabsConversion verifies []int32 -> []int conversion reaches
// the API as the right numeric IDs.
func TestHandlerCloseTabsConversion(t *testing.T) {
	mock := newMockTransport(echoResult(nil))
	handler := NewDBusHandler(NewBrowserAPI(mock))

	if err := handler.CloseTabs([]int32{7, 8}); err != nil {
		t.Fatalf("CloseTabs failed: %v", err)
	}
	reqs := mock.sentRequests()
	if len(reqs) != 1 || reqs[0].Method != MethodCloseTabs {
		t.Fatalf("unexpected requests: %+v", reqs)
	}
}

// TestHandlerOpenTab verifies OpenTab returns the first opened tab's IDs.
func TestHandlerOpenTab(t *testing.T) {
	mock := newMockTransport(echoResult([]TabData{{WindowID: 5, TabID: 99}}))
	handler := NewDBusHandler(NewBrowserAPI(mock))

	win, tab, err := handler.OpenTab("https://example.com")
	if err != nil {
		t.Fatalf("OpenTab failed: %v", err)
	}
	if win != 5 || tab != 99 {
		t.Errorf("got window=%d tab=%d, want 5/99", win, tab)
	}
}
