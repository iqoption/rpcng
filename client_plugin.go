package rpcng

import (
	"io"

	"github.com/iqoption/rpcng/plugin"
)

type (
	// Before start
	ClientBeforeStart interface {
		BeforeStart(addr string) error
	}

	// After start
	ClientAfterStart interface {
		AfterStart(addr string) error
	}

	// Before stop
	ClientBeforeStop interface {
		BeforeStop(addr string) error
	}

	// After stop
	ClientAfterStop interface {
		AfterStop(addr string) error
	}

	// Dial to server
	ClientDialServer interface {
		DialServer(addr string, conn io.ReadWriteCloser, err error) error
	}

	// Client connected
	ClientConnected interface {
		Connected(addr string, conn io.ReadWriteCloser) error
	}

	// Client disconnected
	ClientDisconnected interface {
		Disconnected(addr string, err error) error
	}

	// Client plugins registry
	clientPluginRegistry struct {
		beforeStart  []ClientBeforeStart
		afterStart   []ClientAfterStart
		beforeStop   []ClientBeforeStop
		afterStop    []ClientAfterStop
		dialServer   []ClientDialServer
		connected    []ClientConnected
		disconnected []ClientDisconnected
	}
)

// Add plugins to registry
func (r *clientPluginRegistry) Add(p plugin.Plugin) {
	// before start
	if v, ok := p.(ClientBeforeStart); ok {
		r.beforeStart = append(r.beforeStart, v)
	}

	// after start
	if v, ok := p.(ClientAfterStart); ok {
		r.afterStart = append(r.afterStart, v)
	}

	// before stop
	if v, ok := p.(ClientBeforeStop); ok {
		r.beforeStop = append(r.beforeStop, v)
	}

	// after start
	if v, ok := p.(ClientAfterStop); ok {
		r.afterStop = append(r.afterStop, v)
	}

	// accept connection
	if v, ok := p.(ClientDialServer); ok {
		r.dialServer = append(r.dialServer, v)
	}

	// client connected
	if v, ok := p.(ClientConnected); ok {
		r.connected = append(r.connected, v)
	}

	// client disconnected
	if v, ok := p.(ClientDisconnected); ok {
		r.disconnected = append(r.disconnected, v)
	}
}

// Trigger before start
func (r *clientPluginRegistry) doBeforeStart(addr string) (err error) {
	for _, p := range r.beforeStart {
		if err = p.BeforeStart(addr); err != nil {
			return err
		}
	}

	return nil
}

// Trigger after start
func (r *clientPluginRegistry) doAfterStart(addr string) (err error) {
	for _, p := range r.afterStart {
		if err = p.AfterStart(addr); err != nil {
			return err
		}
	}

	return nil
}

// Trigger before stop
func (r *clientPluginRegistry) doBeforeStop(addr string) (err error) {
	for _, p := range r.beforeStop {
		if err = p.BeforeStop(addr); err != nil {
			return err
		}
	}

	return nil
}

// Trigger after stop
func (r *clientPluginRegistry) doAfterStop(addr string) (err error) {
	for _, p := range r.afterStop {
		if err = p.AfterStop(addr); err != nil {
			return err
		}
	}

	return nil
}

// Trigger dial server
func (r *clientPluginRegistry) doDialServer(addr string, conn io.ReadWriteCloser, err error) error {
	for _, p := range r.dialServer {
		if err := p.DialServer(addr, conn, err); err != nil {
			return err
		}
	}

	return nil
}

// Trigger connected
func (r *clientPluginRegistry) doConnected(addr string, conn io.ReadWriteCloser) (err error) {
	for _, p := range r.connected {
		if err = p.Connected(addr, conn); err != nil {
			return err
		}
	}

	return nil
}

// Trigger disconnected
func (r *clientPluginRegistry) doDisconnected(addr string, err error) error {
	for _, p := range r.disconnected {
		if err := p.Disconnected(addr, err); err != nil {
			return err
		}
	}

	return nil
}
