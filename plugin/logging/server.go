package logging

import (
	"io"
	"net"

	"github.com/iqoption/rpcng/logger"
	"github.com/iqoption/rpcng/plugin"
)

// Server plugin
type serverPlugin struct {
	logger logger.Logger
}

// Constructor
func NewServer(logger logger.Logger) plugin.Plugin {
	return &serverPlugin{
		logger: logger,
	}
}

// Name
func (*serverPlugin) Name() string {
	return "logging"
}

// After start
func (p *serverPlugin) BeforeStart(listener net.Listener, _ []string) error {
	p.logger.Info(
		"Starting Server",
		"protocol", listener.Addr().Network(),
		"listen_addr", listener.Addr().String(),
	)

	return nil
}

// After start
func (p *serverPlugin) AfterStart(listener net.Listener, _ []string) error {
	p.logger.Info(
		"Server started",
		"protocol", listener.Addr().Network(),
		"listen_addr", listener.Addr().String(),
	)

	return nil
}

// Before stop
func (p *serverPlugin) BeforeStop(listener net.Listener, _ []string) error {
	p.logger.Info("Stopping Server")

	return nil
}

// Before stop
func (p *serverPlugin) AfterStop(listener net.Listener, _ []string) error {
	p.logger.Info("Server stopped")

	return nil
}

// Accept connection
func (p *serverPlugin) AcceptConnection(conn io.ReadWriteCloser, err error) error {
	if err == nil {
		if tcp, ok := conn.(*net.TCPConn); ok {
			p.logger.Debug("Accept connection", "local_addr", tcp.LocalAddr().String(), "remote_addr", tcp.RemoteAddr().String())

			return nil
		}

		return nil
	}

	p.logger.Error("Accepting error", "error", err)

	return nil
}

// Client connected
func (p *serverPlugin) ClientConnected(conn io.ReadWriteCloser) error {
	if tcp, ok := conn.(*net.TCPConn); ok {
		p.logger.Debug("Accept connection", "local_addr", tcp.LocalAddr().String(), "remote_addr", tcp.RemoteAddr().String())

		return nil
	}

	p.logger.Debug("Client connected")

	return nil
}

// Client disconnected
func (p *serverPlugin) ClientDisconnected(err error) error {
	if err != nil {
		p.logger.Error("Client disconnected", "err", err)

		return nil
	}

	p.logger.Debug("Client disconnected")

	return nil
}
