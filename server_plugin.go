package rpcng

import (
	"io"
	"net"

	"github.com/iqoption/rpcng/plugin"
)

type (
	// Before start
	ServerBeforeStart interface {
		BeforeStart(listener net.Listener, methods []string) error
	}

	// After start
	ServerAfterStart interface {
		AfterStart(listener net.Listener, methods []string) error
	}

	// Before stop
	ServerBeforeStop interface {
		BeforeStop(listener net.Listener, methods []string) error
	}

	// After stop
	ServerAfterStop interface {
		AfterStop(listener net.Listener, methods []string) error
	}

	// Accept connection
	ServerAcceptConnection interface {
		AcceptConnection(conn io.ReadWriteCloser, err error) error
	}

	// Client connected
	ServerClientConnected interface {
		ClientConnected(conn io.ReadWriteCloser) error
	}

	// Client disconnected
	ServerClientDisconnected interface {
		ClientDisconnected(err error) error
	}

	// Server plugins registry
	serverPluginRegistry struct {
		beforeStart []ServerBeforeStart
		afterStart  []ServerAfterStart
		beforeStop  []ServerBeforeStop
		afterStop   []ServerAfterStop
		acceptConn  []ServerAcceptConnection
		clientConn  []ServerClientConnected
		clientDisc  []ServerClientDisconnected
	}
)

// Add plugins to registry
func (r *serverPluginRegistry) Add(p plugin.Plugin) {
	// before start
	if v, ok := p.(ServerBeforeStart); ok {
		r.beforeStart = append(r.beforeStart, v)
	}

	// after start
	if v, ok := p.(ServerAfterStart); ok {
		r.afterStart = append(r.afterStart, v)
	}

	// before stop
	if v, ok := p.(ServerBeforeStop); ok {
		r.beforeStop = append(r.beforeStop, v)
	}

	// after start
	if v, ok := p.(ServerAfterStop); ok {
		r.afterStop = append(r.afterStop, v)
	}

	// accept connection
	if v, ok := p.(ServerAcceptConnection); ok {
		r.acceptConn = append(r.acceptConn, v)
	}

	// client connected
	if v, ok := p.(ServerClientConnected); ok {
		r.clientConn = append(r.clientConn, v)
	}

	// client disconnected
	if v, ok := p.(ServerClientDisconnected); ok {
		r.clientDisc = append(r.clientDisc, v)
	}
}

// Trigger before start
func (r *serverPluginRegistry) doBeforeStart(listener net.Listener, methods []string) (err error) {
	for _, p := range r.beforeStart {
		if err = p.BeforeStart(listener, methods); err != nil {
			return err
		}
	}

	return nil
}

// Trigger after start
func (r *serverPluginRegistry) doAfterStart(listener net.Listener, methods []string) (err error) {
	for _, p := range r.afterStart {
		if err = p.AfterStart(listener, methods); err != nil {
			return err
		}
	}

	return nil
}

// Trigger before stop
func (r *serverPluginRegistry) doBeforeStop(listener net.Listener, methods []string) (err error) {
	for _, p := range r.beforeStop {
		if err = p.BeforeStop(listener, methods); err != nil {
			return err
		}
	}

	return nil
}

// Trigger after stop
func (r *serverPluginRegistry) doAfterStop(listener net.Listener, methods []string) (err error) {
	for _, p := range r.afterStop {
		if err = p.AfterStop(listener, methods); err != nil {
			return err
		}
	}

	return nil
}

// Trigger accept connection
func (r *serverPluginRegistry) doAcceptConnection(conn io.ReadWriteCloser, err error) error {
	for _, p := range r.acceptConn {
		if err := p.AcceptConnection(conn, err); err != nil {
			return err
		}
	}

	return nil
}

// Trigger client connected
func (r *serverPluginRegistry) doClientConnected(conn io.ReadWriteCloser) (err error) {
	for _, p := range r.clientConn {
		if err = p.ClientConnected(conn); err != nil {
			return err
		}
	}

	return nil
}

// Trigger client disconnected
func (r *serverPluginRegistry) doClientDisconnected(err error) error {
	for _, p := range r.clientDisc {
		if err := p.ClientDisconnected(err); err != nil {
			return err
		}
	}

	return nil
}
