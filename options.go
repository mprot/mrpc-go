package mrpc

// ServerOption represents an option which can be used to configure
// an mrpc server.
type ServerOption func(*serverOptions) error

type serverOptions struct {
	interceptors []ServerInterceptor
}

func defaultServerOptions() serverOptions {
	return serverOptions{}
}

func (o *serverOptions) apply(opts []ServerOption) error {
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return err
		}
	}
	return nil
}

// WithServerInterceptor adds an interceptor for method calls on the
// server side. It is possible to add multiple interceptors. In thas
// case they are executed in the order they are provided.
func WithServerInterceptor(interceptor ServerInterceptor) ServerOption {
	return func(o *serverOptions) error {
		if interceptor == nil {
			return optionError("no interceptor specified")
		}
		o.interceptors = append(o.interceptors, interceptor)
		return nil
	}
}
