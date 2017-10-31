package msgp

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/tinylib/msgp/msgp"
)

type msgpStruct struct {
	A int
}

// DecodeMsg implements msgp.Decodable
func (z *msgpStruct) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxvk uint32
	zxvk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxvk > 0 {
		zxvk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "A":
			z.A, err = dc.ReadInt()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z msgpStruct) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "A"
	err = en.Append(0x81, 0xa1, 0x41)
	if err != nil {
		return err
	}
	err = en.WriteInt(z.A)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z msgpStruct) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "A"
	o = append(o, 0x81, 0xa1, 0x41)
	o = msgp.AppendInt(o, z.A)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *msgpStruct) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zbzg uint32
	zbzg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zbzg > 0 {
		zbzg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "A":
			z.A, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z msgpStruct) Msgsize() (s int) {
	s = 1 + 2 + msgp.IntSize
	return
}

// Test encode
func TestCodecMSGP_Encode(t *testing.T) {
	var (
		err    error
		raw    []byte
		codec  = New()
		values = []interface{}{
			msgpStruct{A: 10},
			true,
			complex128(4i),
			complex64(4i),
			float64(math.MaxFloat64),
			float32(math.MaxFloat32),
			18,
			math.MaxInt64,
			math.MaxInt32,
			math.MaxInt16,
			math.MaxInt8,
			"awesome message pack",
			uint(13),
			uint64(math.MaxUint64),
			uint32(math.MaxUint32),
			uint16(math.MaxUint16),
			uint8(math.MaxUint8),
			[]byte{1, 2, 3, 4, 5, 6},
			time.Now(),
			1 * time.Second,
			errors.New("Oops"),
		}
	)

	for _, v := range values {
		raw, err = codec.Encode(v)
		if err != nil {
			t.Fatalf("Can't encode %T", v)
		}

		t.Logf("Encoded %T bytes %v", v, raw)
	}
}

// Test decode
func TestCodecMSGP_Decode(t *testing.T) {
	var (
		err  error
		raws = [][]byte{
			[]byte{129, 161, 65, 10},
			[]byte{195},
			[]byte{216, 4, 0, 0, 0, 0, 0, 0, 0, 0, 64, 16, 0, 0, 0, 0, 0, 0},
			[]byte{215, 3, 0, 0, 0, 0, 64, 128, 0, 0},
			[]byte{203, 127, 239, 255, 255, 255, 255, 255, 255},
			[]byte{202, 127, 127, 255, 255},
			[]byte{18},
			[]byte{211, 127, 255, 255, 255, 255, 255, 255, 255},
			[]byte{210, 127, 255, 255, 255},
			[]byte{209, 127, 255},
			[]byte{127},
			[]byte{180, 97, 119, 101, 115, 111, 109, 101, 32, 109, 101, 115, 115, 97, 103, 101, 32, 112, 97, 99, 107},
			[]byte{13},
			[]byte{207, 255, 255, 255, 255, 255, 255, 255, 255},
			[]byte{206, 255, 255, 255, 255},
			[]byte{205, 255, 255},
			[]byte{204, 255},
			[]byte{196, 6, 1, 2, 3, 4, 5, 6},
			[]byte{199, 12, 5, 0, 0, 0, 0, 89, 223, 41, 103, 31, 74, 134, 229},
			[]byte{210, 59, 154, 202, 0},
			[]byte{164, 79, 111, 112, 115},
		}
		codec      = New()
		interfaces = []interface{}{
			new(msgpStruct),
			new(bool),
			new(complex128),
			new(complex64),
			new(float64),
			new(float32),
			new(int),
			new(int64),
			new(int32),
			new(int16),
			new(int8),
			new(string),
			new(uint),
			new(uint64),
			new(uint32),
			new(uint16),
			new(uint8),
			new([]byte),
			new(time.Time),
			new(time.Duration),
			errors.New(""),
		}
	)

	for k, v := range raws {
		if err = codec.Decode(v, interfaces[k]); err != nil {
			t.Fatalf("Can't decode %T. Err: %s", interfaces[k], err)
		}

		t.Logf("Decoded %T as %+v", interfaces[k], reflect.ValueOf(interfaces[k]).Elem().Interface())
	}
}
