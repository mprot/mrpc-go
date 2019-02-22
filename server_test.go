package mrpc

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mprot/msgpack-go"
)

func TestServerRegisterValidation(t *testing.T) {
	s := newServer(t)

	ensurePanic := func(t *testing.T, expectedMsg string) {
		if msg, ok := recover().(string); !ok || msg != expectedMsg {
			t.Fatalf("unexpected panic message: %s", msg)
		}

	}

	func() {
		defer ensurePanic(t, "missing service name")
		s.Register(ServiceSpec{})
	}()

	func() {
		defer ensurePanic(t, "missing service")
		s.Register(ServiceSpec{
			Name: "my-service",
		})
	}()

	func() {
		spec := ServiceSpec{
			Name:    "my-service",
			Service: struct{}{},
		}
		s.Register(spec)

		defer ensurePanic(t, "service my-service already registered")
		s.Register(spec)
	}()
}

func TestServerExecute(t *testing.T) {
	ctx := context.Background()
	s := newServer(t)

	handler3Called := false
	handler7Called := false
	deadline, hasDeadline := time.Time{}, false
	s.Register(ServiceSpec{
		Name:    "my-service",
		Service: struct{}{},
		Methods: []MethodSpec{
			{
				ID: 3,
				Handler: func(ctx context.Context, svc interface{}, body []byte) ([]byte, error) {
					handler3Called = true
					return []byte("handler result"), nil
				},
			},
			{
				ID: 7,
				Handler: func(ctx context.Context, svc interface{}, body []byte) ([]byte, error) {
					handler7Called = true
					return nil, errors.New("handler error")
				},
			},
			{
				ID: 13,
				Handler: func(ctx context.Context, svc interface{}, body []byte) ([]byte, error) {
					deadline, hasDeadline = ctx.Deadline()
					return nil, nil
				},
			},
		},
	})

	t.Run("method-not-found", func(t *testing.T) {
		resp := s.Execute(ctx, Request{
			Service: "not-existing-service",
			Method:  3,
		})
		if err := ResponseError(resp); err == nil {
			t.Fatal("expected error, got none")
		} else if err.Error() != "method not-existing-service:3 not found" {
			t.Fatalf("unexpected error: %v", err)
		}

		resp = s.Execute(ctx, Request{
			Service: "my-service",
			Method:  1,
		})
		if err := ResponseError(resp); err == nil {
			t.Fatal("expected error, got none")
		} else if err.Error() != "method my-service:1 not found" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("method-called-successfully", func(t *testing.T) {
		resp := s.Execute(ctx, Request{
			Service: "my-service",
			Method:  3,
		})

		if err := ResponseError(resp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		} else if !handler3Called {
			t.Fatal("handler not called")
		} else if !bytes.Equal(resp.Body, []byte("handler result")) {
			t.Fatalf("unexpected return: %+v", resp.Body)
		}
	})

	t.Run("method-called-with-error", func(t *testing.T) {
		resp := s.Execute(ctx, Request{
			Service: "my-service",
			Method:  7,
		})

		if err := ResponseError(resp); err == nil {
			t.Fatal("expected error, got none")
		} else if !handler7Called {
			t.Fatal("handler not called")
		}
	})

	t.Run("method-called-with-deadline", func(t *testing.T) {
		resp := s.Execute(ctx, Request{
			Service: "my-service",
			Method:  13,
			Headers: RequestHeaders{Timeout: uint64(time.Second)},
		})

		if err := ResponseError(resp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		} else if !hasDeadline || deadline.IsZero() {
			t.Fatal("expected deadline, got none")
		}
	})
}

func TestServerServceMPRC(t *testing.T) {
	ctx := context.Background()
	s := newServer(t)

	expectedResult := []byte("expected result")
	handlerCalled := false
	s.Register(ServiceSpec{
		Name:    "my-service",
		Service: struct{}{},
		Methods: []MethodSpec{
			{
				ID: 7,
				Handler: func(ctx context.Context, svc interface{}, args []byte) ([]byte, error) {
					handlerCalled = true
					return expectedResult, nil
				},
			},
		},
	})

	var req bytes.Buffer
	err := msgpack.Encode(&req, &Request{
		Service: "my-service",
		Method:  7,
		Headers: RequestHeaders{Timeout: uint64(time.Minute)},
		Body:    []byte("request body"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp bytes.Buffer
	err = s.ServeMRPC(ctx, &req, &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if !handlerCalled {
		t.Fatal("handler not called")
	}

	var response Response
	if err = msgpack.Decode(&resp, &response); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if err = ResponseError(response); err != nil {
		t.Fatalf("unexpected response error: %v", err)
	} else if !bytes.Equal(response.Body, expectedResult) {
		t.Fatalf("unexpected return: %+v", response.Body)
	}
}

func newServer(t *testing.T) *Server {
	s, err := NewServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return s
}
