package rpcng

import (
	"math/rand"
	"sync/atomic"

	"github.com/iqoption/rpcng/consistenthash"
	"github.com/iqoption/rpcng/registry"
)

// Tagged entry
type tagged struct {
	tag      string
	index    int64
	services registry.Services
	clients  []*Client
	hash     *consistenthash.Map
	idc      map[string]*Client
}

func newTagged(tag string, services registry.Services, clients []*Client, chash bool) *tagged {
	t := &tagged{
		tag:      tag,
		services: services,
		clients:  clients,
	}

	if chash {
		t.idc = make(map[string]*Client)
		s := make([]string, len(services))
		for i := range services {
			s[i] = services[i].Id
			t.idc[s[i]] = clients[i]
		}
		t.hash = consistenthash.New(200, s...)
	}
	return t
}

func (t *tagged) Tag() string {
	return t.tag
}

func (t *tagged) Random() (*Client, error) {
	n := len(t.clients)
	switch {
	case n <= 0:
		return nil, ErrNotFounded
	case n == 1:
		return t.client(0)
	default:
		return t.client(int64(rand.Intn(n)))
	}
}

func (t *tagged) Hashed(key int) (*Client, error) {
	if t.hash == nil || t.hash.IsEmpty() {
		return nil, ErrNotFounded
	}
	return t.idc[t.hash.GetUint64(uint64(key))], nil
}

func (t *tagged) RoundRobin() (*Client, error) {
	var index = atomic.AddInt64(&t.index, 1)
	return t.client(index)
}

func (t *tagged) client(i int64) (*Client, error) {
	if len(t.clients) == 0 {
		return nil, ErrNotFounded
	}
	return t.clients[int(i)%len(t.clients)], nil
}

func (t *tagged) All() []*Client {
	return t.clients
}

func (t *tagged) HasCHash() bool {
	return t.hash != nil
}

func (t *tagged) Services() registry.Services {
	return t.services
}
