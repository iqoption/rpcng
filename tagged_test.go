package rpcng

import "testing"

func TestEmptyTagged(t *testing.T) {
	p := newTagged("asd", nil, nil, true)
	if _, err := p.RoundRobin(); err != ErrNotFounded {
		t.Error("must return error")
		t.Fail()
	}

	if _, err := p.Random(); err != ErrNotFounded {
		t.Error("must return error")
		t.Fail()
	}

	if _, err := p.Hashed(12345); err != ErrNotFounded {
		t.Error("must return error")
		t.Fail()
	}
}
