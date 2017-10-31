package json

import (
	"encoding/json"

	"github.com/iqoption/rpcng/codec"
)

type codecJSON struct{}

func New() codec.Codec {
	return &codecJSON{}
}

func (c *codecJSON) Encode(source interface{}) (raw []byte, err error) {
	if raw, err = json.Marshal(source); err != nil {
		return nil, err
	}

	return raw, nil
}

func (c *codecJSON) Decode(raw []byte, target interface{}) (err error) {
	if err = json.Unmarshal(raw, &target); err != nil {
		return err
	}

	return nil
}
