package rpcng

import (
	"testing"

	"github.com/iqoption/rpcng/registry"
)

func TestCHash(t *testing.T) {

	services := []*registry.Service{{Id: "asd"}, {Id: "bcd"}, {Id: "lsd"}}

	p := newTagged("tag", services, []*Client{&Client{}, &Client{}, &Client{}}, true)

	for i := 0; i < 10000; i++ {
		a, _ := p.Hashed(i)
		b, _ := p.Hashed(i)
		if a != b {
			t.FailNow()
		}
	}

}

func BenchmarkCHash(b *testing.B) {

	services := []*registry.Service{{Id: "asd"}, {Id: "bcd"}, {Id: "lsd"}}

	t := newTagged("tag", services, []*Client{nil, nil, nil}, true)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.Hashed(i)
	}

}
