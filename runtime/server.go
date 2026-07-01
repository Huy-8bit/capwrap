package capwrap

import (
	"context"
	"errors"
	"net"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

// Server hosts a single bootstrap capability and serves it to every client that
// connects. Generated RegisterXxxServer functions install the capability; user
// code normally only calls NewServer and ListenAndServe.
type Server struct {
	opts      serverOptions
	bootstrap capnp.Client
}

// NewServer returns a Server ready to have a service registered on it.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{}
	for _, o := range opts {
		o(&s.opts)
	}
	return s
}

// SetBootstrap installs the capability served to connecting clients. It is
// called by generated RegisterXxxServer helpers; application code rarely needs
// it directly.
func (s *Server) SetBootstrap(c capnp.Client) {
	if s.bootstrap.IsValid() {
		s.bootstrap.Release()
	}
	s.bootstrap = c
}

// ListenAndServe listens on a TCP address and serves until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return WrapError(err)
	}
	return s.Serve(ctx, ln)
}

// Serve accepts connections on ln until ctx is cancelled or ln is closed.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	if !s.bootstrap.IsValid() {
		return Errorf("no service registered; call a generated RegisterXxxServer function first")
	}

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return WrapError(err)
		}
		go func() { _ = s.ServeConn(ctx, conn) }()
	}
}

// ServeConn serves the registered capability over a single connection. It
// returns when the connection or ctx is done.
func (s *Server) ServeConn(ctx context.Context, rwc net.Conn) error {
	conn := rpc.NewConn(rpc.NewStreamTransport(rwc), &rpc.Options{
		BootstrapClient: s.bootstrap.AddRef(),
	})
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return conn.Close()
	}
}
