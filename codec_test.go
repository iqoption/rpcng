package rpcng

import (
	"testing"

	"github.com/iqoption/rpcng/codec"
	"github.com/iqoption/rpcng/codec/easyjson"
	"github.com/iqoption/rpcng/codec/json"
	"github.com/iqoption/rpcng/codec/msgp"
	"github.com/iqoption/rpcng/codec/xdr"
)

func benchmarkCodecEncode(b *testing.B, c codec.Codec) {
	b.SetParallelism(250)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var err error
		for i := 0; pb.Next(); i++ {
			if _, err = c.Encode(&sample); err != nil {
				b.Fatalf("Can't encode: %s", err)
			}
		}
	})
}

func benchmarkCodecDecode(b *testing.B, c codec.Codec) {
	var (
		raw []byte
		err error
	)

	if raw, err = c.Encode(&sample); err != nil {
		b.Fatalf("Can't encode: %s", err)
	}

	b.SetParallelism(250)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			var (
				err error
				val CodecStruct
			)

			if err = c.Decode(raw, &val); err != nil {
				b.Fatalf("Can't decode: %s", err)
			}
		}
	})
}

func BenchmarkCodecEasyjsonEncode(b *testing.B) {
	benchmarkCodecEncode(b, easyjson.New())
}

func BenchmarkCodecEasyjsonDecode(b *testing.B) {
	benchmarkCodecDecode(b, easyjson.New())
}

func BenchmarkCodecJsonEncode(b *testing.B) {
	benchmarkCodecEncode(b, json.New())
}

func BenchmarkCodecJsonDecode(b *testing.B) {
	benchmarkCodecDecode(b, json.New())
}

func BenchmarkCodecXdrEncode(b *testing.B) {
	benchmarkCodecEncode(b, xdr.New())
}

func BenchmarkCodecXdrDecode(b *testing.B) {
	benchmarkCodecDecode(b, xdr.New())
}

func BenchmarkCodecMsgpEncode(b *testing.B) {
	benchmarkCodecEncode(b, msgp.New())
}

func BenchmarkCodecMsgpDecode(b *testing.B) {
	benchmarkCodecDecode(b, msgp.New())
}
