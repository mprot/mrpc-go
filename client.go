package mrpc

import (
	"context"
	"io"
	"time"

	"github.com/mprot/msgpack-go"
)

// Caller defines an interface for calling remote functions.
type Caller interface {
	Call(ctx context.Context, req Request) (Response, error)
}

// Client is mrpc client to call service methods. A client is transport
// independent.
type Client struct {
	rw io.ReadWriter
}

// NewClient creates a new mrpc client. When calling a method with Call, the
// request will be written to the writing part and the response will be read
// from the reading part of rw.
func NewClient(rw io.ReadWriter) *Client {
	return &Client{rw: rw}
}

// Call calls a remote method by writing the request to the client's writer
// and reading the response from the client's reader.
func (c *Client) Call(ctx context.Context, req Request) (Response, error) {
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 && (req.Headers.Timeout == 0 || uint64(timeout) < req.Headers.Timeout) {
			req.Headers.Timeout = uint64(timeout)
		}
	}

	if err := msgpack.Encode(c.rw, &req); err != nil {
		return Response{}, err
	}

	var resp Response
	err := msgpack.Decode(c.rw, &resp)
	return resp, err
}
