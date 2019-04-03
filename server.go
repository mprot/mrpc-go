package mrpc

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/mprot/msgpack-go"
)

// Registry defines an interface for registering a service specification.
type Registry interface {
	Register(svc ServiceSpec)
}

// Handler is a function type for mrpc handlers. A handler is called for each
// request and basically defines a service's method.
type Handler func(ctx context.Context, svc interface{}, body []byte) ([]byte, error)

// MethodSpec holds the data for a single method. A method has a unique
// id within the defined service and a handler which completes all
// incoming requests.
type MethodSpec struct {
	ID      int
	Handler Handler
}

// ServiceSpec holds the data for a service. A service has a unique name
// and holds the specification for all containing methods.
type ServiceSpec struct {
	Name    string
	Service interface{}
	Methods []MethodSpec
}

// Server is a mrpc server, where services can be registered. A server is
// transport independent and the network layer has to be implemented separately.
type Server struct {
	services  map[string]struct{} // set of service names
	methods   map[string]method   // method id => method
	intercept ServerInterceptor
}

// NewServer creates a new mrpc server with the given options.
func NewServer(o ...ServerOption) (*Server, error) {
	opts := defaultServerOptions()
	if err := opts.apply(o); err != nil {
		return nil, err
	}

	return &Server{
		services:  make(map[string]struct{}),
		methods:   make(map[string]method),
		intercept: serverInterceptorChain(opts.interceptors),
	}, nil
}

// Register registers a service to the server. If an invalid service
// specification is passed or a service with the same id was already
// registered, the function will panic.
func (s *Server) Register(svc ServiceSpec) {
	if svc.Name == "" {
		panic("missing service name")
	} else if svc.Service == nil {
		panic("missing service")
	} else if _, has := s.services[svc.Name]; has {
		panic("service " + svc.Name + " already registered")
	}

	for _, m := range svc.Methods {
		s.methods[methodKey(svc.Name, m.ID)] = method{
			svc:     svc.Service,
			handler: m.Handler,
		}
	}
	s.services[svc.Name] = struct{}{}
}

// Execute executes a single request and calls the corresponding method.
// If the requested service or method was not registered, an error response
// will be returned.
func (s *Server) Execute(ctx context.Context, req Request) Response {
	key := methodKey(req.Service, req.Method)
	method, has := s.methods[key]
	if !has {
		return ErrorResponsef(NotFound, "method %s not found", key)
	}

	call := CallInfo{
		Service: method.svc,
		Method:  key,
		Body:    req.Body,
	}

	cancel := func() {}
	if req.Headers.Timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Headers.Timeout))
	}

	resp, err := s.intercept(ctx, call, method.handler)
	cancel()
	if err != nil {
		return ErrorResponse(err)
	}
	return Response{Body: resp}
}

// ServeMRPC serves a request read from r and writes the response back to w.
// This function only returns an error, if the encoding of the response fails.
// All other errors are encoded in the response and written to w. The reader
// should provide a single request only.
func (s *Server) ServeMRPC(ctx context.Context, r io.Reader, w io.Writer) error {
	var (
		req  Request
		resp Response
	)

	if err := msgpack.Decode(r, &req); err == nil {
		resp = s.Execute(ctx, req)
	} else {
		resp = ErrorResponsef(Unknown, "decode request: %s", err.Error())
	}
	return msgpack.Encode(w, &resp)
}

type method struct {
	svc     interface{}
	handler Handler
}

func methodKey(svc string, method int) string {
	return svc + ":" + strconv.FormatInt(int64(method), 10)
}
