# Cap'n Proto vs gRPC (and where capwrap fits)

capwrap does **not** try to replace gRPC. It makes Cap'n Proto RPC feel familiar
to Go developers who already know gRPC, so you can reach for Cap'n Proto when it
is the better technical fit without paying the full ergonomic cost.

## When gRPC is the better choice

- You need a large, mature ecosystem: load balancing, service mesh integration,
  deadlines/retries/interceptors, tracing, and multi-language clients today.
- Your team already runs gRPC in production and the tooling investment is done.
- You mostly do small unary request/response calls at high concurrency, where
  gRPC-Go is extremely well optimized.

## When Cap'n Proto is worth it

- **Zero-copy / partial reads.** With large structured payloads you can read
  only the fields you need without deserializing the whole message. This is the
  standout win for big records and batch payloads.
- **Capability-based security.** You pass around capabilities (object
  references) rather than ambient authority.
- **Promise pipelining.** You can call a method on the result of another call
  before the first has returned, cutting round trips.
- **Compact wire format** with no separate parse step.

## What capwrap wraps and what it keeps

capwrap generates a thin, gRPC-like layer:

- `Dial<Service>` returns a typed client whose methods take `context.Context`
  and plain Go request structs.
- `Register<Service>Server` lets you implement the service as an ordinary Go
  type returning `(response, error)`.

The raw `capnproto.org/go/capnp/v3` generated types stay reachable. When you
need promise pipelining or capability arguments, drop down to them directly —
capwrap is an on-ramp, not a wall.

## MVP limitations

The MVP generator maps scalar and text/data fields. Methods using lists, nested
structs, unions, or capability parameters are emitted as "not supported" server
stubs and left off the typed client, so the rest of a service still generates
cleanly. Growing this subset is the main path to `v0.1.0`.
