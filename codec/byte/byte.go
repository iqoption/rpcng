package byte

import "errors"

type (
	Encoder interface {
		Encode() ([]byte, error)
	}

	Decoder interface {
		Decode(raw []byte) error
	}

	codecByte struct{}
)

func (*codecByte) Encode(source interface{}) ([]byte, error) {
	var (
		e  Encoder
		ok bool
	)

	if e, ok = source.(Encoder); !ok {
		return nil, errors.New("source should implement Encoder interface")
	}

	return e.Encode()
}

func (*codecByte) Decode(raw []byte, target interface{}) error {
	var (
		d  Decoder
		ok bool
	)

	if d, ok = target.(Decoder); !ok {
		return errors.New("target should implement Decoder interface")
	}

	return d.Decode(raw)
}
