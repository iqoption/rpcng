package xdr

import (
	"github.com/davecgh/go-xdr/xdr"

	"github.com/iqoption/rpcng/codec"
)

type codecXDR struct{}

func New() codec.Codec {
	return &codecXDR{}
}

func (c *codecXDR) Encode(source interface{}) (raw []byte, err error) {
	if raw, err = xdr.Marshal(&source); err != nil {
		return nil, err
	}

	return raw, nil
}

func (c *codecXDR) Decode(raw []byte, target interface{}) (err error) {
	if _, err = xdr.Unmarshal(raw, &target); err != nil {
		return err
	}

	return nil
}
