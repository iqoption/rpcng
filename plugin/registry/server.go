package registry

import (
	"errors"
	"net"

	"github.com/iqoption/rpcng/plugin"
	"github.com/iqoption/rpcng/registry"
)

type serverPlugin struct {
	tags     []string
	service  string
	registry registry.Registry
}

// Constructor
func NewServer(registry registry.Registry, service string, tags []string) plugin.Plugin {
	return &serverPlugin{
		tags:     tags,
		service:  service,
		registry: registry,
	}
}

// Name
func (p *serverPlugin) Name() string {
	return "registry"
}

// After start
func (p *serverPlugin) AfterStart(listener net.Listener, methods []string) error {
	var s = &registry.Service{
		Name: p.service,
	}

	switch listener.Addr().(type) {
	case *net.TCPAddr:
		s.Port = listener.Addr().(*net.TCPAddr).Port
		s.Address = listener.Addr().(*net.TCPAddr).IP.String()
	case *net.UDPAddr:
		s.Port = listener.Addr().(*net.UDPAddr).Port
		s.Address = listener.Addr().(*net.UDPAddr).IP.String()
	default:
		return errors.New("listener addr type is not supported")
	}

	if s.Address == "0.0.0.0" || s.Address == "::" || s.Address == "[::]" {
		return errors.New("invalid service address")
	}

	for _, tag := range p.tags {
		s.Tags = append(s.Tags, tag)
	}

	for _, method := range methods {
		s.Tags = append(s.Tags, method)
	}

	return p.registry.Register(s)
}

// Before stop
func (p *serverPlugin) BeforeStop(listener net.Listener, methods []string) error {
	return p.registry.Deregister(p.service)
}
