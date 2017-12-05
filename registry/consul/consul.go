package consul

import (
	"fmt"
	"net"
	"sort"

	"github.com/hashicorp/consul/api"

	"github.com/iqoption/rpcng/registry"
)

type registryConsul struct {
	client *api.Client
}

// Constructor
func New(host, port string) (consul registry.Registry, err error) {
	var (
		cfg    = api.DefaultConfig()
		client *api.Client
	)

	cfg.Address = net.JoinHostPort(host, port)

	if client, err = api.NewClient(cfg); err != nil {
		return nil, err
	}

	return &registryConsul{
		client: client,
	}, nil
}

// Register
func (c *registryConsul) Register(service *registry.Service) error {
	return c.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      service.Id,
		Name:    service.Name,
		Tags:    service.Tags,
		Port:    service.Port,
		Address: service.Address,
		Checks: api.AgentServiceChecks{
			&api.AgentServiceCheck{
				TCP:      fmt.Sprintf("%s:%d", service.Address, service.Port),
				Timeout:  "1s",
				Interval: "3s",
			},
		},
	})
}

// Deregister
func (c *registryConsul) Deregister(service string) error {
	return c.client.Agent().ServiceDeregister(service)
}

// Services
func (c *registryConsul) Services(service string, tag string) (registry.Services, error) {
	var services, _, err = c.client.Health().Service(service, tag, true, nil)
	if err != nil {
		return nil, err
	}

	var list = make(registry.Services, 0, len(services))
	for _, s := range services {
		// sort
		sort.Strings(s.Service.Tags)

		// append
		list = append(list, &registry.Service{
			Id:      s.Service.ID,
			Name:    s.Service.Service,
			Tags:    s.Service.Tags,
			Port:    s.Service.Port,
			Address: s.Service.Address,
		})
	}

	return list, nil
}
