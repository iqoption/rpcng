package rpcng

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
)

// Client request
type clientRequest struct {
	// vars
	err    error
	skip   bool
	done   chan struct{}
	args   []interface{}
	reply  []interface{}
	client *Client

	// read
	rBuf       *bytes.Buffer
	errLen     uint64
	payloadLen uint64

	// write
	wBuf    *bytes.Buffer
	name    []byte
	nameLen uint64
}

// Constructor
func newClientRequest(client *Client) *clientRequest {
	return &clientRequest{
		// client
		client: client,

		// buffers
		rBuf: bytes.NewBuffer(
			make([]byte, 0, client.RequestBufSize),
		),
		wBuf: bytes.NewBuffer(
			make([]byte, 0, client.RequestBufSize),
		),

		// vars
		done:  make(chan struct{}),
		args:  make([]interface{}, 0),
		reply: make([]interface{}, 0),
	}
}

// Args
func (request *clientRequest) Args(args ...interface{}) *clientRequest {
	request.args = args

	return request
}

// Reply
func (request *clientRequest) Reply(reply ...interface{}) *clientRequest {
	request.reply = reply

	return request
}

// Async
func (request *clientRequest) Call(ctx context.Context) error {
	request.skip = false

	return request.client.call(ctx, request)
}

// Exec
func (request *clientRequest) Send(ctx context.Context) error {
	request.skip = true

	return request.client.send(ctx, request)
}

// Read
func (request *clientRequest) readFrom(reader io.Reader) (err error) {
	defer func() {
		request.done <- struct{}{}
	}()

	// read error
	if err = binary.Read(reader, binary.BigEndian, &request.errLen); err != nil {
		return err
	}

	if request.errLen > 0 {
		request.rBuf.Grow(int(request.errLen))
		if _, err = io.CopyN(request.rBuf, reader, int64(request.errLen)); err != nil {
			return err
		}

		request.err = errors.New(request.rBuf.String())
	}

	// read payload
	if err = binary.Read(reader, binary.BigEndian, &request.payloadLen); err != nil {
		return err
	}

	request.rBuf.Reset()
	request.rBuf.Grow(int(request.payloadLen))
	if _, err = io.CopyN(request.rBuf, reader, int64(request.payloadLen)); err != nil {
		return err
	}

	// decode reply
	var (
		buf  = &bytes.Buffer{}
		size uint64
	)

	for _, v := range request.reply {
		if err = binary.Read(request.rBuf, binary.BigEndian, &size); err != nil {
			return err
		}

		buf.Reset()
		buf.Grow(int(size))

		if _, err = io.CopyN(buf, request.rBuf, int64(size)); err != nil {
			return err
		}

		if err = request.client.Codec.Decode(buf.Bytes(), v); err != nil {
			return err
		}
	}

	return nil
}

// Write
func (request *clientRequest) writeTo(writer io.Writer) (err error) {
	// write method name
	if err = binary.Write(request.wBuf, binary.BigEndian, request.nameLen); err != nil {
		return err
	}

	if _, err = request.wBuf.Write(request.name); err != nil {
		return err
	}

	// write payload
	var raw []byte
	for _, v := range request.args {
		if raw, err = request.client.Codec.Encode(v); err != nil {
			return err
		}

		if err = binary.Write(request.wBuf, binary.BigEndian, uint64(len(raw))); err != nil {
			return err
		}

		if _, err = request.wBuf.Write(raw); err != nil {
			return err
		}
	}

	// write buf
	if err = binary.Write(writer, binary.BigEndian, uint64(request.wBuf.Len())); err != nil {
		return err
	}

	if _, err = request.wBuf.WriteTo(writer); err != nil {
		return err
	}

	return nil
}

// Reset
func (request *clientRequest) reset() *clientRequest {
	// vars
	request.err = nil
	request.skip = false
	request.name = request.name[:0]
	request.args = request.args[:0]
	request.reply = request.reply[:0]
	request.errLen = 0
	request.nameLen = 0
	request.payloadLen = 0

	// buffers
	request.rBuf.Reset()
	request.wBuf.Reset()

	// done
	select {
	case <-request.done:
	default:
	}

	return request
}
