package rpcng

import (
	"io"
	"net"
	"time"

	"github.com/iqoption/rpcng/codec"
)

// TCP Client constructor
func NewTCPClient(addr string, codec codec.Codec) *Client {
	var dialer = &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Minute,
	}

	return &Client{
		Addr: addr,
		Dial: func(addr string) (conn io.ReadWriteCloser, err error) {
			return dialer.Dial("tcp", addr)
		},
		Codec: codec,
	}
}

// TCP Server Constructor
func NewTCPServer(addr string, codec codec.Codec) (*Server, error) {
	var (
		err     error
		tcpAddr *net.TCPAddr
	)

	if tcpAddr, err = net.ResolveTCPAddr("tcp", addr); err != nil {
		return nil, err
	}

	var listener net.Listener
	if listener, err = net.ListenTCP(tcpAddr.Network(), tcpAddr); err != nil {
		return nil, err
	}

	return &Server{
		Codec:    codec,
		listener: listener,
	}, nil
}
