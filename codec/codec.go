package codec

type Codec interface {
	Encode(source interface{}) ([]byte, error)
	Decode(raw []byte, target interface{}) error
}
