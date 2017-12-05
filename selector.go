package rpcng

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/iqoption/rpcng/codec/json"
	"github.com/iqoption/rpcng/registry"
)

var (
	ErrNotFounded = errors.New("can't find any client")
)

type (
	// Selector
	Selector struct {
		// Mutex
		mux sync.RWMutex

		// tagged
		tagged map[string]*tagged

		// Selected
		service string

		// Services registry
		registry registry.Registry

		// Stop chan
		stopChan chan struct{}

		// Discovered tagged
		discovered registry.Services

		// Stop client
		stopClient func(client *Client) error

		// Start client
		startClient func(addr string) (*Client, error)

		// On discovered
		onDiscovered []func(discovered registry.Services)
	}

	// Selected
	Selected struct {
		Hash   string
		Client *Client
	}

	// Tagged entry
	tagged struct {
		index int64
		count int64
		items []*Selected
	}
)

// Constructor
func NewSelector(registry registry.Registry, service string, interval int64) (selector *Selector, err error) {
	selector = &Selector{
		tagged:   make(map[string]*tagged),
		service:  service,
		registry: registry,
		stopChan: make(chan struct{}),

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
		for {
			select {
			case <-ticker.C:
				c.pull() // TODO don't ignore errors
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
	c.mux.Lock()
	defer c.mux.Unlock()

	// close
	close(c.stopChan)

	// free all items
	for tag := range c.tagged {
		for _, wc := range c.tagged[tag].items {
			if err = c.destroySelected(wc); err != nil {
				return err
			}
		}

		delete(c.tagged, tag)
	}

	return nil
}

// Random Client
func (c *Selector) Random(tag string) (*Selected, error) {
	var tagged, err = c.fill(tag)
	if err != nil {
		return nil, err
	}

	if tagged.count <= 0 {
		return nil, fmt.Errorf("can't find client for tag %s", tag)
	}

	var index = atomic.AddInt64(
		&tagged.index,
		rand.Int63n(tagged.count),
	)

	return tagged.items[index%tagged.count], nil
}

// Round robin Client
func (c *Selector) RoundRobin(tag string) (*Selected, error) {
	var tagged, err = c.fill(tag)
	if err != nil {
		return nil, err
	}

	var index = atomic.AddInt64(&tagged.index, 1)

	return tagged.items[index%tagged.count], nil
}

// All items
func (c *Selector) All(tag string) ([]*Selected, error) {
	var tagged, err = c.fill(tag)
	if err != nil {
		return nil, err
	}

	return tagged.items, nil
}

// Fill items
func (c *Selector) fill(tag string) (item *tagged, err error) {
	var founded bool

	c.mux.RLock()
	if item, founded = c.tagged[tag]; founded {
		c.mux.RUnlock()

		if len(item.items) == 0 {
			return nil, ErrNotFounded
		}

		return item, nil
	}

	c.mux.RUnlock()
	c.mux.Lock()

	item = &tagged{
		index: 0,
		items: make([]*Selected, 0, len(c.discovered)),
	}

	var nwc *Selected
	for _, s := range c.discovered {
		for _, v := range s.Tags {
			if v == tag {
				if nwc, err = c.createSelected(s); err != nil {
					c.mux.Unlock()

					return nil, err
				}

				item.items = append(item.items, nwc)
			}
		}
	}

	c.tagged[tag] = item
	c.tagged[tag].count = int64(len(item.items))

	c.mux.Unlock()

	if len(item.items) == 0 {
		return nil, ErrNotFounded
	}

	return item, nil
}

// Pull items
func (c *Selector) pull() (err error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var discovered registry.Services
	if discovered, err = c.registry.Services(c.service, ""); err != nil {
		return err
	}

	var (
		index           int
		newDiscovered   registry.Services
		deletedServices = make(map[string]bool)
	)

	// updated, added
	for i := 0; i < len(discovered); i++ {
		index = -1

		// find
		for j := 0; j < len(c.discovered); j++ {
			if c.isEqualServices(discovered[i], c.discovered[j]) {
				index = j
				break
			}
		}

		// equal
		if index > -1 {
			continue
		}

		// append
		newDiscovered = append(newDiscovered, discovered[i])
	}

	// founded
	for i := 0; i < len(c.discovered); i++ {
		index = -1

		// find
		for j := 0; j < len(discovered); j++ {
			if c.isEqualServices(c.discovered[i], discovered[j]) {
				index = j
				break
			}
		}

		if -1 == index {
			deletedServices[c.hashService(c.discovered[i])] = true
		}
	}

	// replace
	c.discovered = discovered

	// update
	var founded bool
	for tag, tagged := range c.tagged {
		// reset
		tagged.index = 0

		// delete
		for i := 0; i < len(tagged.items); {
			if _, founded = deletedServices[tagged.items[i].Hash]; !founded {
				i++
				continue
			}

			c.destroySelected(tagged.items[i])

			if i == len(tagged.items)-1 {
				tagged.items = tagged.items[:i]
				tagged.count--
				continue
			}

			tagged.items[i] = tagged.items[len(tagged.items)-1]
			tagged.items = tagged.items[:len(tagged.items)-1]
			tagged.count--
		}

		// append
		var service *Selected
		for _, s := range newDiscovered {
			for _, v := range s.Tags {
				if v == tag {
					if service, err = c.createSelected(s); err != nil {
						return err
					}

					tagged.items = append(tagged.items, service)
					tagged.count++
					continue
				}
			}
		}
	}

	// trigger
	for _, fn := range c.onDiscovered {
		fn(c.discovered)
	}

	return nil
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
