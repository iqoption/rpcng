package easyjson

import (
	"errors"

	"github.com/mailru/easyjson"

	"github.com/iqoption/rpcng/codec"
)

type codecEasyjson struct{}

func New() codec.Codec {
	return &codecEasyjson{}
}

func (c *codecEasyjson) Encode(source interface{}) (raw []byte, err error) {
	var (
		ok        bool
		marshaler easyjson.Marshaler
	)

	if marshaler, ok = source.(easyjson.Marshaler); !ok {
		return nil, errors.New("source should implement easyjson.Marshaler interface")
	}

	return easyjson.Marshal(marshaler)
}

func (c *codecEasyjson) Decode(raw []byte, target interface{}) (err error) {
	var (
		ok          bool
		unmarshaler easyjson.Unmarshaler
	)

	if unmarshaler, ok = target.(easyjson.Unmarshaler); !ok {
		return errors.New("source should implement easyjson.Unmarshaler interface")
	}

	return easyjson.Unmarshal(raw, unmarshaler)
}
