package dbus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

type Server struct {
	conn       *dbus.Conn
	browser    string
	handler    BrowserHandler
	props      *prop.Properties
}

type BrowserHandler interface {
	ListTabs() ([]TabInfo, error)
	ActivateTab(tabID string) error
	CloseTab(tabID string) error
	OpenTab(url string) (string, error)
}

func NewServer(browser string, handler BrowserHandler) (*Server, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	s := &Server{
		conn:    conn,
		browser: browser,
		handler: handler,
	}

	return s, nil
}

func (s *Server) Start() error {
	serviceName := ServiceName(s.browser)
	objectPath := ObjectPath(s.browser)

	// Request service name
	reply, err := s.conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to request name %s: %w", serviceName, err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("name %s already taken", serviceName)
	}

	// Export methods
	err = s.conn.Export(s, objectPath, InterfaceBrowser)
	if err != nil {
		return fmt.Errorf("failed to export object: %w", err)
	}

	// Export introspection
	introspectionXML := generateIntrospection()
	err = s.conn.Export(introspect.Introspectable(introspectionXML), objectPath,
		"org.freedesktop.DBus.Introspectable")
	if err != nil {
		return fmt.Errorf("failed to export introspection: %w", err)
	}

	// Initialize properties (for future use)
	propsSpec := map[string]map[string]*prop.Prop{
		InterfaceBrowser: {
			"BrowserName": {
				Value:    s.browser,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}

	props, err := prop.Export(s.conn, objectPath, propsSpec)
	if err != nil {
		return fmt.Errorf("failed to export properties: %w", err)
	}
	s.props = props

	// D-Bus server started successfully
	return nil
}

func (s *Server) Stop() error {
	if s.conn != nil {
		serviceName := ServiceName(s.browser)
		s.conn.ReleaseName(serviceName)
		return s.conn.Close()
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

func (s *Server) ActivateTab(tabID string) (bool, *dbus.Error) {
	err := s.handler.ActivateTab(tabID)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	return true, nil
}

func (s *Server) CloseTab(tabID string) (bool, *dbus.Error) {
	err := s.handler.CloseTab(tabID)
	if err != nil {
		return false, dbus.MakeFailedError(err)
	}
	return true, nil
}

func (s *Server) OpenTab(url string) (string, *dbus.Error) {
	tabID, err := s.handler.OpenTab(url)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}
	return tabID, nil
}

func generateIntrospection() string {
	return `
<node>
	<interface name="dev.slastra.TabCtl.Browser">
		<method name="ListTabs">
			<arg direction="out" type="a(sssibb)" />
		</method>
		<method name="ActivateTab">
			<arg direction="in" type="s" name="tab_id" />
			<arg direction="out" type="b" name="success" />
		</method>
		<method name="CloseTab">
			<arg direction="in" type="s" name="tab_id" />
			<arg direction="out" type="b" name="success" />
		</method>
		<method name="OpenTab">
			<arg direction="in" type="s" name="url" />
			<arg direction="out" type="s" name="tab_id" />
		</method>
		<property name="BrowserName" type="s" access="read" />
	</interface>
</node>`
}