package registry

type (
	Service struct {
		Name    string
		Tags    []string
		Port    int
		Address string
	}

	Services = []*Service

	Registry interface {
		Register(service *Service) error
		Deregister(name string) error
		Services(service, tag string) (Services, error)
	}
)
