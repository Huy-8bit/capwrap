package capwrap

import (
	"context"
	"net"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

// ClientConn is a client-side RPC connection. Generated DialXxx wrappers turn
// its bootstrap capability into a typed, gRPC-like client.
type ClientConn struct {
	conn *rpc.Conn
}

// Dial opens a TCP connection to addr and prepares it for RPC.
func Dial(ctx context.Context, addr string, opts ...DialOption) (*ClientConn, error) {
	var o dialOptions
	for _, opt := range opts {
		opt(&o)
	}

	rawConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, WrapError(err)
	}

	conn := rpc.NewConn(rpc.NewStreamTransport(rawConn), nil)
	return &ClientConn{conn: conn}, nil
}

// Bootstrap returns the remote bootstrap capability. Generated DialXxx wrappers
// convert it into a typed client; application code rarely calls it directly.
func (c *ClientConn) Bootstrap(ctx context.Context) capnp.Client {
	return c.conn.Bootstrap(ctx)
}

// Close tears down the connection.
func (c *ClientConn) Close() error {
	return c.conn.Close()
}
