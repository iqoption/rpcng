package keepalive

import (
	"io"
	"net"
	"time"

	"github.com/iqoption/rpcng/plugin"
)

type serverPlugin struct {
	keepAlivePeriod int
}

// DefaultKeepAlivePeriod is the default keep alive period in seconds
const DefaultKeepAlivePeriod = 30

// Constructor
// Keep alive period in seconds
// If value greater then 0, Server use keep alive if negative Server not used keep alive connections
// Default is DefaultKeepAlivePeriod
func NewServer(keepAlivePeriod int) plugin.Plugin {
	if keepAlivePeriod == 0 {
		keepAlivePeriod = DefaultKeepAlivePeriod
	}

	return &serverPlugin{
		keepAlivePeriod: keepAlivePeriod,
	}
}

// Name
func (p *serverPlugin) Name() string {
	return "keepalive"
}

// Accept connection
func (p *serverPlugin) ClientConnected(conn io.ReadWriteCloser) error {
	if p.keepAlivePeriod >= 0 {
		return nil
	}

	tcp, isTCP := conn.(*net.TCPConn)
	if !isTCP {
		return nil
	}

	if err := tcp.SetKeepAlive(true); err != nil {
		return err
	}

	if err := tcp.SetKeepAlivePeriod(time.Duration(p.keepAlivePeriod) * time.Second); err != nil {
		return err
	}

	return nil
}
