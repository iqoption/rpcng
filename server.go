package rpcng

import (
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/iqoption/rpcng/codec"
	"github.com/iqoption/rpcng/plugin"
)

// Server
type Server struct {
	// The maximum number of concurrent rpc calls the Server may perform
	// Default is DefaultConcurrency
	Concurrency int

	// The default requests buffer sizes
	// Default is DefaultRequestBufSize
	RequestBufSize int

	// The maximum number of pending responses in the queue
	// Default is DefaultPendingResponses
	PendingResponses int

	// The Codec see Codec interface
	Codec codec.Codec

	// The Server obtains new Client connections via listener.Accept()
	listener net.Listener

	// Stop wait group
	stopWg sync.WaitGroup

	// Allowed methods
	methods serverMethods

	// Plugin registry
	plugins serverPluginRegistry

	// Stopping chain
	stopChan chan struct{}

	// Workers chain
	workerChain chan struct{}

	// Requests pool
	requestPool sync.Pool
}

const (
	// DefaultConcurrency is the default number of concurrent calls the Server can process
	DefaultConcurrency = 32 * 1024

	// DefaultPendingResponses is the default number of pending responses handled by Server
	DefaultPendingResponses = 32 * 1024
)

// Handler handler
func (s *Server) Handler(handler ServerHandler) (err error) {
	if s.stopChan != nil {
		return errors.New("can't add handler to already started server")
	}

	var methods = handler.Methods()
	for name, callback := range methods {
		if err = s.methods.add(name, callback, handler); err != nil {
			return err
		}
	}

	return nil
}

// Add plugins
func (s *Server) Plugin(plugins ...plugin.Plugin) {
	for _, p := range plugins {
		s.plugins.Add(p)
	}
}

// Blocking version of Start
func (s *Server) Serve() (err error) {
	if err = s.Start(); err != nil {
		return err
	}

	s.stopWg.Wait()

	return nil
}

// Start Server
func (s *Server) Start() (err error) {
	// checks
	if s.Codec == nil {
		return errors.New("server should has any codec")
	}

	if s.listener == nil {
		return errors.New("server should has any listener")
	}

	if s.stopChan != nil {
		return errors.New("server already started")
	}

	if len(s.methods.items) == 0 {
		return errors.New("server should have at least one handler")
	}

	// trigger
	if err = s.plugins.doBeforeStart(s.listener, s.methods.names); err != nil {
		return err
	}

	// defaults
	if s.Concurrency == 0 {
		s.Concurrency = DefaultConcurrency
	}

	if s.RequestBufSize == 0 {
		s.RequestBufSize = DefaultRequestBufSize
	}

	if s.PendingResponses == 0 {
		s.PendingResponses = DefaultPendingResponses
	}

	s.stopChan = make(chan struct{})
	s.workerChain = make(chan struct{}, s.Concurrency)
	s.requestPool.New = func() interface{} {
		return newServerRequest(s)
	}

	// start accept connections
	s.stopWg.Add(1)
	go s.acceptConnections()

	// trigger plugins
	if err = s.plugins.doAfterStart(s.listener, s.methods.names); err != nil {
		return err
	}

	return nil
}

// Stop Server
func (s *Server) Stop() (err error) {
	if s.stopChan == nil {
		return errors.New("server must be started before stopping it")
	}

	// trigger plugins
	if err = s.plugins.doBeforeStop(s.listener, s.methods.names); err != nil {
		return err
	}

	defer func() { s.stopChan = nil }()
	close(s.stopChan)

	s.stopWg.Wait()

	// trigger plugins
	if err = s.plugins.doAfterStop(s.listener, s.methods.names); err != nil {
		return err
	}

	return nil
}

// Is Server stopped
func (s *Server) isStopped() bool {
	if nil == s.stopChan {
		return true
	}

	select {
	case <-s.stopChan:
		return true
	default:
		return false
	}
}

// Start accepting Server connections
func (s *Server) acceptConnections() {
	// sync
	defer s.stopWg.Done()

	var (
		aErr error
		pErr error
		conn io.ReadWriteCloser
	)

	for {
		var acceptChan = make(chan struct{})
		go func() {
			conn, aErr = s.listener.Accept()
			close(acceptChan)
		}()

		select {
		case <-s.stopChan:
			s.listener.Close()
			<-acceptChan
			return
		case <-acceptChan:
		}

		// trigger
		if pErr = s.plugins.doAcceptConnection(conn, aErr); aErr == nil && pErr == nil {
			s.stopWg.Add(1)
			go s.handleConnection(conn)
			continue
		}

		// handle error
		select {
		case <-s.stopChan:
			return
		case <-time.After(time.Second):
		}
	}
}

// Handle single connection
func (s *Server) handleConnection(conn io.ReadWriteCloser) {
	// sync
	defer s.stopWg.Done()

	// trigger
	if err := s.plugins.doClientConnected(conn); err != nil {
		return
	}

	// process connection
	var (
		stopChan     = make(chan struct{})
		responseChan = make(chan *serverRequest, s.PendingResponses)
	)

	var doneReaderChan = make(chan error)
	go s.connectionReader(conn, responseChan, stopChan, doneReaderChan)

	var doneWriterChan = make(chan error)
	go s.connectionWriter(conn, responseChan, stopChan, doneWriterChan)

	// wait
	var err error
	select {
	case err = <-doneReaderChan:
		close(stopChan)
		conn.Close()
		<-doneWriterChan
	case err = <-doneWriterChan:
		close(stopChan)
		conn.Close()
		<-doneReaderChan
	case <-s.stopChan:
		close(stopChan)
		conn.Close()
		<-doneReaderChan
		<-doneWriterChan
	}

	// close
	close(doneReaderChan)
	close(doneWriterChan)

	// trigger
	s.plugins.doClientDisconnected(err)
}

// Connection reader
func (s *Server) connectionReader(reader io.Reader, responseChan chan *serverRequest, stopChan <-chan struct{}, doneChan chan<- error) {
	var err error

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic has occured while reading: %v", r)
		}

		doneChan <- err
	}()

	var request *serverRequest

	for {
		select {
		case <-stopChan:
			return
		default:
			runtime.Gosched()

			request = s.requestPool.Get().(*serverRequest)

			if err = request.readFrom(reader); err != nil {
				if s.isStopped() || err == io.ErrUnexpectedEOF || err == io.EOF {
					err = nil
					return
				}

				err = fmt.Errorf("error has occurred while reading: %v", err)
				return
			}

			go s.handleRequest(responseChan, request)
		}
	}
}

// Handle request
func (s *Server) handleRequest(responseChan chan<- *serverRequest, request *serverRequest) {
	s.workerChain <- struct{}{}
	defer func() {
		if request.id == 0 {
			<-s.workerChain
			return
		}

		if r := recover(); r != nil {
			request.err = fmt.Errorf("panic has occurred while handling request: %v", r)
			responseChan <- request
		}

		<-s.workerChain
	}()

	request.err = s.methods.dispatch(request.input, request.output, s.Codec, request.id > 0)

	if request.id == 0 {
		s.requestPool.Put(
			request.reset(),
		)

		return
	}

	responseChan <- request
}

// Connection writer
func (s *Server) connectionWriter(writer io.Writer, responseChan <-chan *serverRequest, stopChan <-chan struct{}, doneChan chan<- error) {
	var err error

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic has occured while writing: %v", r)
		}

		doneChan <- err
	}()

	var request *serverRequest

	for {
		select {
		case <-stopChan:
			return
		case request = <-responseChan:
			runtime.Gosched()

			if err = request.writeTo(writer); err != nil {
				if s.isStopped() {
					err = nil
					return
				}

				err = fmt.Errorf("error has occurred while writing: %v", err)
				return
			}

			s.requestPool.Put(
				request.reset(),
			)
		}
	}
}
