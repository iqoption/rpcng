package logging

import (
	"io"

	"github.com/iqoption/rpcng/logger"
	"github.com/iqoption/rpcng/plugin"
)

// Client plugin
type clientPlugin struct {
	logger logger.Logger
}

// Constructor
func NewClient(logger logger.Logger) plugin.Plugin {
	return &clientPlugin{
		logger: logger,
	}
}

// Name
func (c *clientPlugin) Name() string {
	return "logging"
}

// Before start
func (c *clientPlugin) BeforeStart(addr string) error {
	c.logger.Debug("Starting client", "addr", addr)

	return nil
}

// After start
func (c *clientPlugin) AfterStart(addr string) error {
	c.logger.Debug("Client started", "addr", addr)

	return nil
}

// Before stop
func (c *clientPlugin) BeforeStop(addr string) error {
	c.logger.Debug("Stopping client", "addr", addr)

	return nil
}

// After stop
func (c *clientPlugin) AfterStop(addr string) error {
	c.logger.Debug("Client stopped", "addr", addr)

	return nil
}

// Server dial
func (c *clientPlugin) DialServer(addr string, conn io.ReadWriteCloser, err error) error {
	if err != nil {
		c.logger.Error("Can't establish connection to Server", "addr", addr, "err", err)
	}

	return nil
}

// Connected
func (c *clientPlugin) Connected(addr string, conn io.ReadWriteCloser) error {
	c.logger.Debug("Connected to server", "addr", addr)

	return nil
}

// Disconnected
func (c *clientPlugin) Disconnected(addr string, err error) error {
	if err == nil {
		c.logger.Debug("Disconnected from server", "addr", addr)

		return nil
	}

	c.logger.Debug("Disconnected from server", "addr", addr, "err", err)

	return nil
}
