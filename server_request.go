package rpcng

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Request
type serverRequest struct {
	// vars
	id  uint64
	err error

	// read
	input      *bytes.Buffer
	payloadLen uint64

	// write
	output *bytes.Buffer
	errLen uint64
	errBuf []byte
}

// Constructor
func newServerRequest(server *Server) *serverRequest {
	return &serverRequest{
		input: bytes.NewBuffer(
			make([]byte, 0, server.RequestBufSize),
		),
		output: bytes.NewBuffer(
			make([]byte, 0, server.RequestBufSize),
		),
	}
}

// Read
func (r *serverRequest) readFrom(reader io.Reader) (err error) {
	// id
	if err = binary.Read(reader, binary.BigEndian, &r.id); err != nil {
		return err
	}

	// payload
	if err = binary.Read(reader, binary.BigEndian, &r.payloadLen); err != nil {
		return err
	}

	// read payload
	if _, err = io.CopyN(r.input, reader, int64(r.payloadLen)); err != nil {
		return err
	}

	return nil
}

// Write
func (r *serverRequest) writeTo(writer io.Writer) (err error) {
	if err = binary.Write(writer, binary.BigEndian, r.id); err != nil {
		return err
	}

	// err
	if r.err != nil {
		r.errBuf = []byte(r.err.Error())
		r.errLen = uint64(len(r.errBuf))
	}

	if err = binary.Write(writer, binary.BigEndian, r.errLen); err != nil {
		return err
	}

	if r.errLen > 0 { // not write empty
		if _, err = writer.Write(r.errBuf); err != nil {
			return err
		}
	}

	// payload
	if err = binary.Write(writer, binary.BigEndian, uint64(r.output.Len())); err != nil {
		return
	}

	if r.output.Len() > 0 { // not write empty
		if _, err = r.output.WriteTo(writer); err != nil {
			return err
		}
	}

	return nil
}

// Reset
func (r *serverRequest) reset() *serverRequest {
	// vars
	r.id = 0
	r.err = nil

	// read
	r.input.Reset()
	r.payloadLen = 0

	// write
	r.output.Reset()
	r.errLen = 0
	r.errBuf = r.errBuf[:0]

	return r
}
