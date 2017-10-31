package rpcng

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/iqoption/rpcng/codec"
)

// Method
type (
	serverMethod struct {
		iNum  int
		fVal  reflect.Value
		fArgs []reflect.Type
	}

	serverMethods struct {
		mux   sync.Mutex
		names []string
		items map[string]*serverMethod
	}
)

// Add
func (m *serverMethods) add(name, callback string, handler ServerHandler) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if nil == m.items {
		m.items = make(map[string]*serverMethod)
	}

	var (
		hVal   = reflect.ValueOf(handler)
		method = &serverMethod{}
	)

	if hVal.Kind() != reflect.Ptr {
		return errors.New("only ptr handlers are allowed")
	}

	method.fVal = hVal.MethodByName(callback)
	if !method.fVal.IsValid() {
		return fmt.Errorf(`name "%s" not founded`, name)
	}

	method.iNum = method.fVal.Type().NumIn()

	for i := 0; i < method.iNum; i++ {
		method.fArgs = append(method.fArgs, method.fVal.Type().In(i))
	}

	var oNum = method.fVal.Type().NumOut()
	if oNum == 0 {
		m.names = append(m.names, name)
		m.items[name] = method

		return nil
	}

	var (
		lOut = method.fVal.Type().Out(oNum - 1)
		eTyp = reflect.TypeOf((*error)(nil)).Elem()
	)

	if !lOut.Implements(eTyp) {
		return errors.New("last output parameter always should be error")
	}

	m.names = append(m.names, name)
	m.items[name] = method

	return nil
}

// Dispatch
func (m *serverMethods) dispatch(input io.Reader, output io.Writer, codec codec.Codec, reply bool) (err error) {
	var methodLen uint64
	if err = binary.Read(input, binary.BigEndian, &methodLen); err != nil {
		return err
	}

	var name = &bytes.Buffer{}
	if _, err = io.CopyN(name, input, int64(methodLen)); err != nil {
		return err
	}

	var (
		ok     bool
		method *serverMethod
	)

	if method, ok = m.items[name.String()]; !ok {
		return fmt.Errorf(`name "%s" is not registered`, name)
	}

	var args = make([]interface{}, 0, method.iNum)
	for _, arg := range method.fArgs {
		args = append(args, reflect.New(arg).Interface())
	}

	var (
		argLen uint64
		argBuf = &bytes.Buffer{}
	)

	for _, arg := range args {
		argBuf.Reset()

		if err = binary.Read(input, binary.BigEndian, &argLen); err != nil {
			return err
		}

		if _, err = io.CopyN(argBuf, input, int64(argLen)); err != nil {
			return err
		}

		if err = codec.Decode(argBuf.Bytes(), arg); err != nil {
			return err
		}
	}

	var requestValues = make([]reflect.Value, 0, method.iNum)
	for _, val := range args {
		requestValues = append(requestValues, reflect.ValueOf(val).Elem())
	}

	// call
	var response []reflect.Value
	if reply {
		response = method.fVal.Call(requestValues)
	} else {
		method.fVal.Call(requestValues)
	}

	// no results
	if len(response) == 0 {
		return nil
	}

	// if err != nil
	var lastIndex = len(response) - 1
	if len(response) == 1 && !response[lastIndex].IsNil() {
		return response[lastIndex].Interface().(error)
	}

	// if error == nil
	if len(response) == 1 && response[lastIndex].IsNil() {
		return nil
	}

	var (
		valBuf []byte
		valLen uint64
	)

	for i := 0; i < len(response)-1; i++ {
		argBuf.Reset()

		if valBuf, err = codec.Encode(response[i].Interface()); err != nil {
			return err
		}

		valLen = uint64(len(valBuf))

		if err = binary.Write(output, binary.BigEndian, valLen); err != nil {
			return err
		}

		if _, err = output.Write(valBuf); err != nil {
			return err
		}
	}

	// if response exist and err != nil
	if !response[lastIndex].IsNil() {
		return response[lastIndex].Interface().(error)
	}

	// if response exist and err == nil
	return nil
}
