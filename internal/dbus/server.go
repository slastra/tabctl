package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

type Server struct {
	conn    *dbus.Conn
	browser string
	// instance is the bus-name element actually claimed by Start ("Chrome",
	// "Chrome2", ...). Empty until Start succeeds.
	instance string
	handler  BrowserHandler
}

// BrowserHandler is the mediator-side implementation the server adapts.
type BrowserHandler interface {
	ListTabs() ([]TabInfo, error)
	ActivateTab(tabID int32, focused bool) error
	CloseTabs(tabIDs []int32) error
	OpenTab(url string) (windowID, tabID int32, err error)
	GetInfo() Info
}

func NewServer(browser string, handler BrowserHandler) (*Server, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}
	return &Server{conn: conn, browser: browser, handler: handler}, nil
}

// requestNameFunc matches (*dbus.Conn).RequestName so claimName is testable
// without a session bus.
type requestNameFunc func(name string, flags dbus.RequestNameFlags) (dbus.RequestNameReply, error)

// claimName takes the first free bus name for this browser.
//
// One mediator process is spawned per browser *profile*, and a profile is
// invisible from this side: Chrome passes only the extension origin in argv
// and every profile shares one browser process, so all of them derive the
// same browser name. Insisting on that one name let only a single profile
// serve at a time (issue #2); instead each mediator walks the instance names
// until one is free. RequestName is atomic on the bus, so two mediators
// racing for the same slot cannot both win it.
func claimName(request requestNameFunc, browser string) (string, error) {
	for n := 1; n <= MaxInstances; n++ {
		instance := InstanceName(browser, n)
		reply, err := request(ServiceName(instance), dbus.NameFlagDoNotQueue)
		if err != nil {
			return "", fmt.Errorf("failed to request name %s: %w", ServiceName(instance), err)
		}
		if reply == dbus.RequestNameReplyPrimaryOwner {
			return instance, nil
		}
	}
	return "", fmt.Errorf("all %d %s instance names are already taken on the session bus",
		MaxInstances, browser)
}

func (s *Server) Start() error {
	instance, err := claimName(s.conn.RequestName, s.browser)
	if err != nil {
		return err
	}
	// Recorded before the exports so a partially-started server still
	// releases its name on Stop.
	s.instance = instance
	objectPath := ObjectPath(instance)

	if err := s.conn.Export(s, objectPath, InterfaceBrowser); err != nil {
		return fmt.Errorf("failed to export object: %w", err)
	}

	introspectionXML := generateIntrospection()
	if err := s.conn.Export(introspect.Introspectable(introspectionXML), objectPath,
		"org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	return nil
}

// Name returns the bus-name element this server claimed ("Chrome",
// "Chrome2", ...). Empty before Start succeeds.
func (s *Server) Name() string { return s.instance }

func (s *Server) Stop() error {
	if s.conn != nil && s.instance != "" {
		s.conn.ReleaseName(ServiceName(s.instance))
		// Don't close the connection — dbus.SessionBus() returns a shared
		// process-level singleton. Closing it would break other callers.
	}
	return nil
}

// D-Bus method implementations

func (s *Server) ListTabs() ([]TabInfo, *dbus.Error) {
	tabs, err := s.handler.ListTabs()
	if err != nil {
		return nil, dbus.MakeFailedError(err)
	}
	return tabs, nil
}

func (s *Server) ActivateTab(tabID int32, focused bool) *dbus.Error {
	if err := s.handler.ActivateTab(tabID, focused); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func (s *Server) CloseTabs(tabIDs []int32) *dbus.Error {
	if err := s.handler.CloseTabs(tabIDs); err != nil {
		return dbus.MakeFailedError(err)
	}
	return nil
}

func (s *Server) OpenTab(url string) (int32, int32, *dbus.Error) {
	windowID, tabID, err := s.handler.OpenTab(url)
	if err != nil {
		return 0, 0, dbus.MakeFailedError(err)
	}
	return windowID, tabID, nil
}

func (s *Server) GetInfo() (string, string, int32, int32, bool, *dbus.Error) {
	i := s.handler.GetInfo()
	return i.MediatorVersion, i.ExtensionVersion, i.MediatorProtocol, i.ExtensionProtocol, i.Compatible, nil
}

// EmitTabsUpdated broadcasts the TabsUpdated signal: the browser's tab set
// changed and subscribers should re-pull ListTabs. Carries no payload.
func (s *Server) EmitTabsUpdated() {
	if s.instance == "" {
		return // not started; nothing is listening on our path yet
	}
	_ = s.conn.Emit(ObjectPath(s.instance), InterfaceBrowser+".TabsUpdated")
}

func generateIntrospection() string {
	return `
<node>
	<interface name="dev.slastra.TabCtl.Browser">
		<method name="ListTabs">
			<arg direction="out" type="a(iissibb)" />
		</method>
		<method name="ActivateTab">
			<arg direction="in" type="i" name="tab_id" />
			<arg direction="in" type="b" name="focused" />
		</method>
		<method name="CloseTabs">
			<arg direction="in" type="ai" name="tab_ids" />
		</method>
		<method name="OpenTab">
			<arg direction="in" type="s" name="url" />
			<arg direction="out" type="i" name="window_id" />
			<arg direction="out" type="i" name="tab_id" />
		</method>
		<method name="GetInfo">
			<arg direction="out" type="s" name="mediator_version" />
			<arg direction="out" type="s" name="extension_version" />
			<arg direction="out" type="i" name="mediator_protocol" />
			<arg direction="out" type="i" name="extension_protocol" />
			<arg direction="out" type="b" name="compatible" />
		</method>
		<signal name="TabsUpdated" />
	</interface>
</node>`
}
