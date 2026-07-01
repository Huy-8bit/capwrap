// Package capwrap is the small runtime that generated *.capwrap.go files depend
// on. It hides the boilerplate of setting up Cap'n Proto RPC connections behind
// a gRPC-like Dial / Serve surface, while leaving the raw capnp types reachable
// for advanced use.
package capwrap

// serverOptions holds tuning for a Server. It is intentionally empty for the
// MVP; new knobs (interceptors, transports, logging) are added here without
// breaking the ServerOption signature.
type serverOptions struct{}

// ServerOption configures a Server created with NewServer.
type ServerOption func(*serverOptions)

// dialOptions holds tuning for a client connection.
type dialOptions struct{}

// DialOption configures a client connection created with Dial.
type DialOption func(*dialOptions)
