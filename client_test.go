package mrpc

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/mprot/msgpack-go"
)

func TestClientCall(t *testing.T) {
	request := Request{
		Service: "service",
		Method:  3,
		Headers: RequestHeaders{},
		Body:    []byte("request body"),
	}

	response := Response{
		ErrorCode: OK,
		ErrorText: "error",
		Body:      []byte("response body"),
	}

	t.Run("no-timeout", func(t *testing.T) {
		ctx := context.Background()
		conn := newClientConn(func(req Request) Response {
			if !reflect.DeepEqual(req, request) {
				t.Fatalf("unexpected request: %#v", req)
			}
			return response
		})
		client := NewClient(conn)
		resp, err := client.Call(ctx, request)
		switch {
		case err != nil:
			t.Fatalf("unexpected error: %v", err)
		case !reflect.DeepEqual(resp, response):
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("header-timeout", func(t *testing.T) {
		request.Headers.Timeout = 100

		ctx := context.Background()
		conn := newClientConn(func(req Request) Response {
			if !reflect.DeepEqual(req, request) {
				t.Fatalf("unexpected request: %#v", req)
			}
			return response
		})
		client := NewClient(conn)
		resp, err := client.Call(ctx, request)
		switch {
		case err != nil:
			t.Fatalf("unexpected error: %v", err)
		case !reflect.DeepEqual(resp, response):
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("context-timeout", func(t *testing.T) {
		request.Headers.Timeout = 0

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn := newClientConn(func(req Request) Response {
			if req.Headers.Timeout == 0 {
				t.Fatal("unexpected timeout: 0")
			}

			req.Headers.Timeout = 0
			if !reflect.DeepEqual(req, request) {
				t.Fatalf("unexpected request: %#v", req)
			}
			return response
		})
		client := NewClient(conn)
		resp, err := client.Call(ctx, request)
		switch {
		case err != nil:
			t.Fatalf("unexpected error: %v", err)
		case !reflect.DeepEqual(resp, response):
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("header-and-context-timeout", func(t *testing.T) {
		// prefer timeout header
		request.Headers.Timeout = uint64(time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		conn := newClientConn(func(req Request) Response {
			if req.Headers.Timeout != uint64(time.Second) {
				t.Fatalf("unexpected timeout: %v", time.Duration(req.Headers.Timeout))
			}
			if !reflect.DeepEqual(req, request) {
				t.Fatalf("unexpected request: %#v", req)
			}
			return response
		})
		client := NewClient(conn)
		resp, err := client.Call(ctx, request)
		switch {
		case err != nil:
			t.Fatalf("unexpected error: %v", err)
		case !reflect.DeepEqual(resp, response):
			t.Fatalf("unexpected response: %#v", resp)
		}
		cancel()

		// prefer context timeout
		request.Headers.Timeout = uint64(time.Minute)

		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		conn = newClientConn(func(req Request) Response {
			if req.Headers.Timeout == 0 || req.Headers.Timeout > uint64(time.Minute) {
				t.Fatalf("unexpected timeout: %v", time.Duration(req.Headers.Timeout))
			}

			req.Headers.Timeout = request.Headers.Timeout
			if !reflect.DeepEqual(req, request) {
				t.Fatalf("unexpected request: %#v", req)
			}
			return response
		})
		client = NewClient(conn)
		resp, err = client.Call(ctx, request)
		switch {
		case err != nil:
			t.Fatalf("unexpected error: %v", err)
		case !reflect.DeepEqual(resp, response):
			t.Fatalf("unexpected response: %#v", resp)
		}
		cancel()
	})
}

type clientConn struct {
	f    func(Request) Response
	req  []byte
	resp []byte
	pos  int
}

func newClientConn(f func(Request) Response) *clientConn {
	return &clientConn{f: f, pos: -1}
}

func (c *clientConn) Read(p []byte) (int, error) {
	if c.pos < 0 {
		var req Request
		err := msgpack.Decode(bytes.NewReader(c.req), &req)
		if err != nil {
			return 0, err
		}
		resp := c.f(req)

		var buf bytes.Buffer
		if err := msgpack.Encode(&buf, &resp); err != nil {
			return 0, err
		}
		c.resp = buf.Bytes()
	}
	n := copy(p, c.resp)
	c.pos += n
	return n, nil
}

func (c *clientConn) Write(p []byte) (int, error) {
	c.req = append(c.req, p...)
	return len(p), nil
}
