package mrpc

import (
	"context"
)

// CallInfo holds details about a method call on the server side.
type CallInfo struct {
	Service interface{}
	Method  string
	Body    []byte
}

// ServerInterceptor defines a function type for intercepting a request on
// the server side. The interceptor is responsible to call h to complete the
// method call.
type ServerInterceptor func(ctx context.Context, call CallInfo, h Handler) ([]byte, error)

func serverInterceptorChain(interceptors []ServerInterceptor) ServerInterceptor {
	switch len(interceptors) {
	case 0:
		return func(ctx context.Context, call CallInfo, h Handler) ([]byte, error) {
			return h(ctx, call.Service, call.Body)
		}

	case 1:
		return interceptors[0]

	default:
		return func(ctx context.Context, call CallInfo, h Handler) ([]byte, error) {
			var next Handler

			idx := 0
			next = func(ctx context.Context, svc interface{}, body []byte) ([]byte, error) {
				if idx++; idx == len(interceptors) {
					return h(ctx, call.Service, call.Body)
				}
				return interceptors[idx](ctx, call, next)
			}

			return interceptors[idx](ctx, call, next)
		}
	}
}
