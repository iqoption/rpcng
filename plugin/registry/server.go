package registry

import (
	"errors"
	"net"

	"github.com/iqoption/rpcng/plugin"
	"github.com/iqoption/rpcng/registry"
)

type serverPlugin struct {
	service  registry.Service
	registry registry.Registry
}

// Constructor
func NewServer(registry registry.Registry, service registry.Service) plugin.Plugin {
	return &serverPlugin{
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
	switch listener.Addr().(type) {
	case *net.TCPAddr:
		p.service.Port = listener.Addr().(*net.TCPAddr).Port
		p.service.Address = listener.Addr().(*net.TCPAddr).IP.String()
	case *net.UDPAddr:
		p.service.Port = listener.Addr().(*net.UDPAddr).Port
		p.service.Address = listener.Addr().(*net.UDPAddr).IP.String()
	default:
		return errors.New("listener addr type is not supported")
	}

	if p.service.Address == "0.0.0.0" || p.service.Address == "::" || p.service.Address == "[::]" {
		return errors.New("invalid service address")
	}

	for _, tag := range p.service.Tags {
		p.service.Tags = append(p.service.Tags, tag)
	}

	for _, method := range methods {
		p.service.Tags = append(p.service.Tags, method)
	}

	return p.registry.Register(&p.service)
}

// Before stop
func (p *serverPlugin) BeforeStop(listener net.Listener, methods []string) error {
	return p.registry.Deregister(p.service.Id)
}
