package rpcng

//msgpack:gen
//easyjson:json
type CodecStruct struct {
	A int64
	B int32
	C string
	D []int64
	E map[string]string
}

var sample = CodecStruct{
	A: 13,
	B: 28,
	C: "sample",
	D: []int64{1, 2, 3},
	E: map[string]string{
		"a": "b",
		"c": "d",
	},
}
