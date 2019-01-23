package rpcng

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iqoption/rpcng/codec"
	"github.com/iqoption/rpcng/plugin"
)

// Client
type Client struct {
	// Server address
	Addr string

	// The Client calls this callback when it needs new connection to the Server
	Dial func(addr string) (conn io.ReadWriteCloser, err error)

	// The Codec see Codec interface
	Codec codec.Codec

	// The number of concurrent connections the Client should establish to the sever
	// Default is DefaultConnections
	Connections int

	// The maximum number of pending requests in the queue.
	//
	// The number of pending requests should exceed the expected number
	// of concurrent goroutines calling Client's methods.
	// Otherwise a lot of ClientError.Overflow errors may appear.
	//
	// Default is DefaultPendingRequests.
	PendingRequests int

	// The default requests buffer sizes
	// Default is DefaultRequestBufSize
	RequestBufSize int

	// Stop wait group
	stopWg sync.WaitGroup

	// Plugins registry
	plugins clientPluginRegistry

	// Stop chan
	stopChan chan struct{}

	// Request pool
	requestPool sync.Pool

	// Request chan
	requestsChan chan *clientRequest

	// Requests count
	requestsCount uint64
}

var (
	ErrClientStopped = errors.New("client stopped")
)

const (
	// The number of concurrent connections the Client should establish to the sever
	DefaultConnections = 1

	// DefaultPendingRequests is the default number of pending requests handled by Client
	DefaultPendingRequests = 32 * 1024

	// DefaultRequestBufSize is a default requests buffer size
	DefaultRequestBufSize = 1024
)

// Add plugins
func (c *Client) Plugin(plugins ...plugin.Plugin) {
	for _, p := range plugins {
		c.plugins.Add(p)
	}
}

// Start Client. Establishes connection to the Server
func (c *Client) Start() error {
	// check
	if c.Addr == "" {
		return errors.New("client should has any Server address")
	}

	if c.Dial == nil {
		return errors.New("client should has any dialer func")
	}

	if c.Codec == nil {
		return errors.New("client should has any codec")
	}

	if c.stopChan != nil {
		return errors.New("client already started")
	}

	// trigger
	if err := c.plugins.doBeforeStart(c.Addr); err != nil {
		return err
	}

	// defaults
	if c.Connections == 0 {
		c.Connections = DefaultConnections
	}

	if c.PendingRequests == 0 {
		c.PendingRequests = DefaultPendingRequests
	}

	if c.RequestBufSize == 0 {
		c.RequestBufSize = DefaultRequestBufSize
	}

	// chains
	c.stopChan = make(chan struct{})
	c.requestPool.New = func() interface{} {
		return newClientRequest(c)
	}
	c.requestsChan = make(chan *clientRequest, c.PendingRequests)

	// run handlers
	for i := 0; i < c.Connections; i++ {
		c.stopWg.Add(1)
		go c.handleConnections()
	}

	// trigger
	if err := c.plugins.doAfterStart(c.Addr); err != nil {
		return err
	}

	return nil
}

// Stop Client. Stopped Client can be started again
func (c *Client) Stop() error {
	if c.stopChan == nil {
		return errors.New("client must be started before stopping it")
	}

	// trigger
	if err := c.plugins.doBeforeStop(c.Addr); err != nil {
		return err
	}

	defer func() { c.stopChan = nil }()
	close(c.stopChan)

	c.stopWg.Wait()

	// trigger
	if err := c.plugins.doAfterStop(c.Addr); err != nil {
		return err
	}

	return nil
}

// Create new output
func (c *Client) Request(name string) *clientRequest {
	var request = c.requestPool.Get().(*clientRequest)

	request.name = []byte(name)
	request.nameLen = uint64(len(request.name))

	return request
}

// Handle connections
func (c *Client) handleConnections() {
	// sync
	defer c.stopWg.Done()

	// handle connections
	var (
		dErr error
		pErr error
		conn io.ReadWriteCloser
	)

	for {
		dialChan := make(chan struct{})
		go func() {
			conn, dErr = c.Dial(c.Addr)
			close(dialChan)
		}()

		select {
		case <-c.stopChan:
			<-dialChan
			return
		case <-dialChan:
		}

		// trigger
		if pErr = c.plugins.doDialServer(c.Addr, conn, dErr); dErr == nil && pErr == nil {
			c.handleConnection(conn)

			select {
			case <-c.stopChan:
				return
			default:
				continue
			}
		}

		select {
		case <-c.stopChan:
			return
		case <-time.After(time.Second):
		}
	}
}

// Handle connection
func (c *Client) handleConnection(conn io.ReadWriteCloser) {
	// trigger
	if err := c.plugins.doConnected(c.Addr, conn); err != nil {
		return
	}

	// process connection
	var (
		err                 error
		stopChan            = make(chan struct{})
		pendingRequests     = make(map[uint64]*clientRequest)
		pendingRequestsLock = &sync.Mutex{}
	)

	var doneReaderChan = make(chan error)
	go c.connectionReader(conn, pendingRequests, pendingRequestsLock, stopChan, doneReaderChan)

	var doneWriterChan = make(chan error)
	go c.connectionWriter(conn, pendingRequests, pendingRequestsLock, stopChan, doneWriterChan)

	// wait
	select {
	case err = <-doneReaderChan:
		close(stopChan)
		conn.Close()
		<-doneWriterChan
	case err = <-doneWriterChan:
		close(stopChan)
		conn.Close()
		<-doneReaderChan
	case <-c.stopChan:
		close(stopChan)
		conn.Close()
		<-doneReaderChan
		<-doneWriterChan
	}

	for _, r := range pendingRequests {
		r.err = err
		if r.done != nil {
			close(r.done)
		}

		atomic.AddUint64(&c.requestsCount, ^uint64(0))
	}

	c.plugins.doDisconnected(c.Addr, err)
}

// Connection reader
func (c *Client) connectionReader(
	reader io.Reader,
	pendingRequests map[uint64]*clientRequest,
	pendingRequestsLock *sync.Mutex,
	stopChan <-chan struct{},
	doneChan chan<- error,
) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			doneChan <- fmt.Errorf("panic occured: %v", r)
			return
		}

		doneChan <- err
	}()

	var id uint64
	for {
		select {
		case <-stopChan:
			err = ErrClientStopped
			return

		default:
			// read response
			if err = binary.Read(reader, binary.BigEndian, &id); err != nil {
				return
			}

			// remove from pending
			pendingRequestsLock.Lock()
			request, ok := pendingRequests[id]
			if ok {
				delete(pendingRequests, id)
			}
			pendingRequestsLock.Unlock()

			if !ok {
				err = fmt.Errorf("unexpected message %d", id)
				return
			}

			atomic.AddUint64(&c.requestsCount, ^uint64(0))

			// read output
			if err = request.readFrom(reader); err != nil {
				return
			}
		}
	}
}

// Connection writer
func (c *Client) connectionWriter(
	writer io.Writer,
	pendingRequests map[uint64]*clientRequest,
	pendingRequestsLock *sync.Mutex,
	stopChan <-chan struct{},
	doneChan chan<- error,
) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			doneChan <- fmt.Errorf("panic occured: %v", r)
			return
		}

		doneChan <- err
	}()

	var (
		id                   uint64
		request              *clientRequest
		requestId            uint64
		pendingRequestsCount int
	)

	for {
		// get output
		select {
		case <-stopChan:
			err = ErrClientStopped
			return
		case request = <-c.requestsChan:

		}

		// cancelled
		if request.err != nil {
			c.requestPool.Put(
				request.reset(),
			)

			continue
		}

		id = 0

		// add to pending if needle
		if !request.skip {
			// first output
			if requestId == 0 {
				requestId = 1
			}

			// store output
			pendingRequestsLock.Lock()
			pendingRequestsCount = len(pendingRequests)
			for {
				if _, ok := pendingRequests[requestId]; !ok {
					break
				}

				requestId++
			}
			pendingRequests[requestId] = request
			pendingRequestsLock.Unlock()

			atomic.AddUint64(&c.requestsCount, 1)

			if pendingRequestsCount > c.PendingRequests {
				err = fmt.Errorf(
					`the Server "%s" didn't return "%d" responses yet. Closing Server connection in order to prevent Client resource leaks`,
					c.Addr,
					pendingRequestsCount,
				)
				return
			}

			id = requestId
		}

		// wrote
		if err = binary.Write(writer, binary.BigEndian, id); err != nil {
			return
		}

		if err = request.writeTo(writer); err != nil {
			return
		}

		// skipped
		if request.skip {
			c.requestPool.Put(
				request.reset(),
			)
		}
	}
}

// Call output
func (c *Client) call(ctx context.Context, request *clientRequest) (err error) {
	// to queue
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.requestsChan <- request:
	}

	// wait reply
	select {
	case <-ctx.Done():
		request.err = ctx.Err()
	case <-request.done:
	}

	return request.err
}

// Send output
func (c *Client) send(ctx context.Context, request *clientRequest) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.requestsChan <- request:
	}

	return nil
}
