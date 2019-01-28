package rpcng

import (
	"errors"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/iqoption/rpcng/codec/json"
	"github.com/iqoption/rpcng/logger"
	"github.com/iqoption/rpcng/registry"
)

var ErrNotFounded = errors.New("can't find any client")

// Selector
type Selector struct {
	// Selected
	service string

	// Services registry
	registry registry.Registry

	// tagged (clients pools by tags)
	tmu    sync.RWMutex
	tagged map[string]*tagged

	// Mutex
	mux sync.RWMutex

	// Stop chan
	stopChan chan struct{}
	stopped  chan struct{}

	// Discovered tagged
	discovered registry.Services

	// Stop client
	stopClient func(client *Client) error

	// Start client
	startClient func(addr string) (*Client, error)

	// On discovered
	onDiscovered []func(discovered registry.Services)

	log logger.Logger
}

// Constructor
func NewSelector(registry registry.Registry, service string, interval int64, log logger.Logger) (selector *Selector, err error) {
	selector = &Selector{
		service:  service,
		registry: registry,
		stopChan: make(chan struct{}),
		stopped:  make(chan struct{}),
		tagged:   make(map[string]*tagged),
		log:      log,

		// defaults
		startClient: func(addr string) (client *Client, err error) {
			client = NewTCPClient(addr, json.New())
			if err = client.Start(); err != nil {
				return nil, err
			}

			return client, nil
		},
		stopClient: func(client *Client) error {
			return client.Stop()
		},
	}

	if err = selector.pull(); err != nil {
		return nil, err
	}

	var ticker = time.NewTicker(time.Duration(interval) * time.Second)
	go func(c *Selector) {
		defer close(c.stopped)
		for {
			select {
			case <-ticker.C:
				err := c.pull() // TODO don't ignore errors
				c.log.Error("Error lookup services", "error", err)
			case <-c.stopChan:
				ticker.Stop()
				return
			}
		}
	}(selector)

	return selector, nil
}

// Start Client
func (c *Selector) StartClient(fn func(addr string) (*Client, error)) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.startClient = fn
}

// Stop Client
func (c *Selector) StopClient(fn func(client *Client) error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.stopClient = fn
}

// On discovery
func (c *Selector) OnDiscovered(fn func(discovered registry.Services)) {
	c.mux.Lock()
	defer c.mux.Unlock()

	fn(c.discovered)
	c.onDiscovered = append(c.onDiscovered, fn)
}

// Stop
func (c *Selector) Stop() (err error) {
	// close
	close(c.stopChan)
	<-c.stopped

	c.tmu.Lock()
	defer c.tmu.Unlock()
	// close all clients
	for _, p := range c.tagged {
		for _, cl := range p.All() {
			c.stop(cl)
		}
	}

	// discard map
	c.tagged = make(map[string]*tagged)

	return nil
}

// Random Client
func (c *Selector) Random(tag string) (*Client, error) {
	var p, err = c.fill(tag, false)
	if err != nil {
		return nil, err
	}

	return p.Random()
}

// Random Client
func (c *Selector) Hashed(tag string, key int) (*Client, error) {
	var p, err = c.fill(tag, true)
	if err != nil {
		return nil, err
	}

	return p.Hashed(key)
}

// Round robin Client
func (c *Selector) RoundRobin(tag string) (*Client, error) {
	var p, err = c.fill(tag, false)
	if err != nil {
		return nil, err
	}
	return p.RoundRobin()
}

// All items
func (c *Selector) All(tag string) ([]*Client, error) {
	var p, err = c.fill(tag, false)
	if err != nil {
		return nil, err
	}

	return p.All(), nil
}

// Fill items, chash (consistent hashing flag)
func (c *Selector) fill(tag string, chash bool) (*tagged, error) {
	// fast path
	c.tmu.RLock()
	if p, ok := c.tagged[tag]; ok && (!chash || p.HasCHash()) {
		c.tmu.RUnlock()
		return p, nil
	}
	c.tmu.RUnlock()

	// slow path
	c.mux.Lock() // only one tag can be created at one moment
	defer c.mux.Unlock()

	// double check
	c.tmu.RLock()
	p, ok := c.tagged[tag]
	if ok && (!chash || p.HasCHash()) {
		c.tmu.RUnlock()
		return p, nil
	}
	c.tmu.RUnlock()

	s := c.getServices(tag, c.discovered, nil)
	if !chash && p != nil {
		chash = p.HasCHash()
	}
	return c.replaceTagged(tag, p, s, chash)
}

func (c *Selector) getDiscovered() registry.Services {
	c.mux.RLock()
	d := c.discovered
	c.mux.RUnlock()
	return d
}

func (c *Selector) getServices(tag string, discovered, buf registry.Services) registry.Services {
	buf = buf[:0]
	for _, s := range discovered {
		for _, v := range s.Tags {
			if v == tag {
				buf = append(buf, s)
			}
		}
	}
	return buf
}

// Pull items
func (c *Selector) pull() (err error) {

	var discovered registry.Services
	if discovered, err = c.registry.Services(c.service, ""); err != nil {
		return err
	}
	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].Id < discovered[j].Id

	})

	if c.sameServices(c.getDiscovered(), discovered) {
		return nil
	}

	c.mux.Lock()
	// replace
	c.discovered = discovered
	// trigger // do we ever need it ? remove ?
	for _, fn := range c.onDiscovered {
		fn(c.discovered)
	}

	c.mux.Unlock()

	var tbuf = make([]*tagged, 0, 1024)
	c.tmu.RLock()
	for _, t := range c.tagged {
		tbuf = append(tbuf, t)
	}
	c.tmu.RUnlock()

	var buf = make([]*registry.Service, 128)
	for _, p := range tbuf {
		s := c.getServices(p.Tag(), discovered, buf)
		if c.sameServices(p.Services(), s) {
			continue
		}

		// copy from buf
		ns := make(registry.Services, len(s))
		copy(ns, s)

		if _, err := c.replaceTagged(p.Tag(), p, ns, p.HasCHash()); err != nil {
			c.log.Error("failure to replace tagged", "error", err)
		}
	}

	return nil
}

// replaceTagged replaces old instance (can be null) with new one based on old clients, finalize unneeded clients
// and returns new one
func (c *Selector) replaceTagged(tag string, old *tagged, s registry.Services, chash bool) (*tagged, error) {
	p, err := c.newTagged(tag, old, s, chash)
	if err != nil {
		return nil, err
	}

	c.tmu.Lock()
	c.tagged[tag] = p
	c.tmu.Unlock()

	if old == nil {
		return p, nil
	}

	// close not needed clients
	var os = old.Services()
	for i := range os {
		found := false
		for j := range s {
			if os[i].Id == s[j].Id {
				found = true
				break
			}
		}
		if !found {
			c.stop(old.All()[i])
		}
	}
	return p, nil
}

// newTagged creates tagged instance with reusing clients from old instance (can be null)
func (c *Selector) newTagged(tag string, old *tagged, s registry.Services, chash bool) (*tagged, error) {
	// for cleanup in case of errors
	created := make([]*Client, len(s))

	ncl := make([]*Client, len(s))

	var os registry.Services
	if old != nil {
		os = old.Services()
	}

	var err error
	for i := range s {
		// copy existed clients
		for j := range os {
			if os[j].Id == s[i].Id {
				ncl[i] = old.All()[j]
				break
			}
		}
		// create new
		if ncl[i] == nil {
			ncl[i], err = c.connect(s[i])
			if err != nil {
				err = errors.New("failed to open client to " + s[i].Id + ":" + err.Error())
				break
			}
			created = append(created, ncl[i])
		}
	}

	if err != nil {
		// clean up
		for i := range created {
			c.stop(created[i])
		}
		return nil, err
	}

	return newTagged(tag, s, ncl, chash), nil
}

func (c *Selector) sameServices(a, b registry.Services) bool {
	if len(a) != len(b) {
		return false
	}

	// both slice already sorted by Id
	for i := range a {
		if a[i].Id != b[i].Id {
			return false
		}
	}

	return true
}

func (c *Selector) connect(s *registry.Service) (*Client, error) {
	var addr = net.JoinHostPort(s.Address, strconv.FormatInt(int64(s.Port), 10))

	return c.startClient(addr)
}

func (c *Selector) stop(cl *Client) error {
	if cl == nil {
		return nil
	}
	// TODO await completition + timeout
	return c.stopClient(cl)
}
