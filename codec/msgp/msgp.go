package msgp

import (
	"errors"
	"fmt"
	"time"

	"github.com/tinylib/msgp/msgp"

	"github.com/iqoption/rpcng/codec"
)

type codecMSGP struct{}

func New() codec.Codec {
	return &codecMSGP{}
}

func (c *codecMSGP) Encode(source interface{}) (raw []byte, err error) {
	switch v := source.(type) {
	case msgp.Marshaler:
		return v.MarshalMsg(raw)
	case bool:
		raw = msgp.Require(raw, msgp.BoolSize)
		return msgp.AppendBool(raw, v), nil
	case complex128:
		raw = msgp.Require(raw, msgp.Complex128Size)
		return msgp.AppendComplex128(raw, v), nil
	case complex64:
		raw = msgp.Require(raw, msgp.Complex64Size)
		return msgp.AppendComplex64(raw, v), nil
	case float64:
		raw = msgp.Require(raw, msgp.Float64Size)
		return msgp.AppendFloat64(raw, v), nil
	case float32:
		raw = msgp.Require(raw, msgp.Float32Size)
		return msgp.AppendFloat32(raw, v), nil
	case int:
		raw = msgp.Require(raw, msgp.IntSize)
		return msgp.AppendInt(raw, v), nil
	case int64:
		raw = msgp.Require(raw, msgp.Int64Size)
		return msgp.AppendInt64(raw, v), nil
	case int32:
		raw = msgp.Require(raw, msgp.Int32Size)
		return msgp.AppendInt32(raw, v), nil
	case int8:
		raw = msgp.Require(raw, msgp.Int8Size)
		return msgp.AppendInt8(raw, v), nil
	case string:
		raw = msgp.Require(raw, msgp.StringPrefixSize+len(v))
		return msgp.AppendString(raw, v), nil
	case uint:
		raw = msgp.Require(raw, msgp.UintSize)
		return msgp.AppendUint(raw, v), nil
	case uint64:
		raw = msgp.Require(raw, msgp.Uint64Size)
		return msgp.AppendUint64(raw, v), nil
	case uint32:
		raw = msgp.Require(raw, msgp.Uint32Size)
		return msgp.AppendUint32(raw, v), nil
	case uint16:
		raw = msgp.Require(raw, msgp.Uint16Size)
		return msgp.AppendUint16(raw, v), nil
	case uint8:
		raw = msgp.Require(raw, msgp.Uint8Size)
		return msgp.AppendUint8(raw, v), nil
	case []byte:
		raw = msgp.Require(raw, msgp.BytesPrefixSize+len(v))
		return msgp.AppendBytes(raw, v), nil
	case time.Time:
		raw = msgp.Require(raw, msgp.TimeSize)
		return msgp.AppendTime(raw, v), nil
	case time.Duration:
		raw = msgp.Require(raw, msgp.Int64Size)
		return msgp.AppendInt64(raw, int64(v)), nil
	case error:
		raw = msgp.Require(raw, msgp.StringPrefixSize+len(v.Error()))
		return msgp.AppendString(raw, v.Error()), nil
	default:
		return nil, fmt.Errorf("source should implement msgp.Marshaler interface but %T given", v)
	}
}

func (c *codecMSGP) Decode(raw []byte, target interface{}) (err error) {
	switch v := target.(type) {
	case msgp.Unmarshaler:
		raw, err = v.UnmarshalMsg(raw)
	case *bool:
		*v, raw, err = msgp.ReadBoolBytes(raw)
	case *complex128:
		*v, raw, err = msgp.ReadComplex128Bytes(raw)
	case *complex64:
		*v, raw, err = msgp.ReadComplex64Bytes(raw)
	case *float64:
		*v, raw, err = msgp.ReadFloat64Bytes(raw)
	case *float32:
		*v, raw, err = msgp.ReadFloat32Bytes(raw)
	case *int:
		*v, raw, err = msgp.ReadIntBytes(raw)
	case *int64:
		*v, raw, err = msgp.ReadInt64Bytes(raw)
	case *int32:
		*v, raw, err = msgp.ReadInt32Bytes(raw)
	case *int16:
		*v, raw, err = msgp.ReadInt16Bytes(raw)
	case *int8:
		*v, raw, err = msgp.ReadInt8Bytes(raw)
	case *string:
		*v, raw, err = msgp.ReadStringBytes(raw)
	case *uint:
		*v, raw, err = msgp.ReadUintBytes(raw)
	case *uint64:
		*v, raw, err = msgp.ReadUint64Bytes(raw)
	case *uint32:
		*v, raw, err = msgp.ReadUint32Bytes(raw)
	case *uint16:
		*v, raw, err = msgp.ReadUint16Bytes(raw)
	case *uint8:
		*v, raw, err = msgp.ReadUint8Bytes(raw)
	case *[]byte:
		*v, raw, err = msgp.ReadBytesBytes(raw, *v)
	case *time.Time:
		*v, raw, err = msgp.ReadTimeBytes(raw)
	case *time.Duration:
		var d int64
		d, raw, err = msgp.ReadInt64Bytes(raw)
		*v = time.Duration(d)
	case error:
		var msg string
		msg, raw, err = msgp.ReadStringBytes(raw)
		v = errors.New(msg)
	default:
		return fmt.Errorf("source should implement msgp.Unmarshaler interface but %T given", v)
	}

	return err
}
