package rpcng

// handler
type ServerHandler interface {
	Methods() map[string]string
}
