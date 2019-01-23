package rpcng

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/iqoption/rpcng/codec/json"
	"github.com/iqoption/rpcng/logger"
	"github.com/iqoption/rpcng/registry"
)

var (
	ErrNotFounded = errors.New("can't find any client")
)

type (
	// Selector
	Selector struct {
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

	// Selected
	Selected struct {
		Hash   string
		Client *Client
	}
)

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
		defer func() { c.stopped <- struct{}{} }()
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
	if p, ok := c.tagged[tag]; ok && (!chash || p.HasCHash()) {
		c.tmu.RUnlock()
		return p, nil
	}
	c.tmu.RUnlock()

	s := c.getServices(tag, c.getDiscovered(), nil)

	clients := make([]*Client, len(s))
	for i := range clients {
		var err error
		clients[i], err = c.connect(s[i])
		if err != nil {
			for i := range clients {
				c.stop(clients[i])
			}
			return nil, err
		}
	}

	p := newTagged(tag, s, clients, chash)

	c.tmu.Lock()
	c.tagged[tag] = p
	c.tmu.Unlock()

	return p, nil
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

	var closing = make([]*Client, 0, 1024)
	var buf = make([]*registry.Service, 128)
loop:
	for _, p := range tbuf {
		s := c.getServices(p.Tag(), discovered, buf)
		if c.sameServices(p.Services(), s) {
			continue
		}
		// copy from buf
		ns := make(registry.Services, len(s))
		copy(ns, s)

		// for cleanup in case of errors
		created := make([]*Client, len(s))

		ncl := make([]*Client, len(s))

		// copy existed clients, create new
		os := p.services
		for i := range ns {
			for j := range os {
				if os[j].Id == ns[i].Id {
					ncl[i] = p.All()[j]
					break
				}
			}
			if ncl[i] == nil {
				var err error
				ncl[i], err = c.connect(ns[i])
				if err != nil {
					c.log.Error("failed open conection", "error", err, "service", ns[i].Id)
					for i := range created {
						c.stop(created[i])
					}
					continue loop
				}
				created = append(created, ncl[i])
			}
		}

		np := newTagged(p.Tag(), ns, ncl, p.HasCHash())

		c.tmu.Lock()
		c.tagged[np.Tag()] = np
		c.tmu.Unlock()

		// close not needed
		for i := range os {
			found := false
			for j := range ns {
				if os[i].Id == ns[j].Id {
					found = true
					break
				}
			}
			if !found {
				closing = append(closing, p.All()[i])
			}
		}
	}

	for i := range closing {
		c.stop(closing[i])
	}

	return nil
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

// Create service
func (c *Selector) createSelected(discovered *registry.Service) (*Selected, error) {
	var (
		err      error
		addr     = net.JoinHostPort(discovered.Address, strconv.FormatInt(int64(discovered.Port), 10))
		selected = &Selected{}
	)

	selected.Hash = c.hashService(discovered)
	if selected.Client, err = c.startClient(addr); err != nil {
		return nil, err
	}

	return selected, nil
}

// Destroy service
func (c *Selector) destroySelected(selected *Selected) (err error) {
	if err = c.stopClient(selected.Client); err != nil {
		return err
	}

	selected.Hash = ""
	selected.Client = nil
	selected = nil

	return nil
}

// Selected Hash
func (c *Selector) hashService(s *registry.Service) string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// Is equal discovered
func (c *Selector) isEqualServices(a, b *registry.Service) bool {
	return reflect.DeepEqual(a, b)
}
