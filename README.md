# capwrap

> **Status: experimental / MVP.** APIs will change. Not production-ready yet.

**capwrap** is a small developer-experience layer for
[Cap'n Proto RPC](https://capnproto.org/) in Go. It generates a thin, gRPC-like
wrapper on top of the official `capnproto.org/go/capnp/v3` runtime so you can:

- define a service in a `.capnp` schema,
- generate a Go interface + server + client wrapper,
- implement the server as a plain Go type, and
- call methods from the client as ordinary Go functions with `context.Context`.

No new protocol is invented — the wire format and runtime are 100% Cap'n Proto.
capwrap only wraps the ergonomics.

## Why not just use gRPC?

capwrap is **not** a gRPC replacement. gRPC has a larger ecosystem and is a
great default. Reach for Cap'n Proto (via capwrap) when you specifically want:

- **zero-copy / partial reads** of large structured payloads,
- **capability-based** RPC (pass object references, not ambient authority),
- **promise pipelining** to remove round trips.

See [docs/capnp-vs-grpc.md](docs/capnp-vs-grpc.md) for the trade-offs.

## Install

```bash
# The wrapper generator
go install github.com/Huy-8bit/capwrap/cmd/capwrap-gen@latest

# The Cap'n Proto compiler + Go plugin (needed to generate the base bindings)
# See https://capnproto.org/install.html
go install capnproto.org/go/capnp/v3/capnpc-go@latest
```

Add the runtime to your module:

```bash
go get github.com/Huy-8bit/capwrap/runtime
```

## Quickstart

### 1. Schema

```capnp
# calculator.capnp
@0xc7d2d2b67fd7ab31;
using Go = import "/go.capnp";
$Go.package("calc");
$Go.import("your/module/calc");

interface Calculator {
  sayHello @0 (name :Text) -> (message :Text);
  add @1 (a :Int64, b :Int64) -> (sum :Int64);
}
```

### 2. Generate

One command produces both the Cap'n Proto bindings and the capwrap wrapper —
`capwrap-gen` runs `capnp compile` for you:

```bash
capwrap-gen calculator.capnp
#  wrote calculator.capwrap.go
#  wrote calculator.capnp.go
```

Pass `-capnp=false` to skip the compile step (e.g. in a Makefile that runs
`capnp` itself), or `-I <dir>` to override the auto-detected std include path.
If `capnp` is not installed, capwrap-gen still writes the wrapper and prints how
to generate the bindings.

#### Prefer protobuf syntax?

`capwrap-gen` also accepts a proto3 `.proto` file. It is translated to a sibling
`.capnp` and everything else is the same:

```proto
// math.proto
syntax = "proto3";
option go_package = "your/module/mathpb;mathpb";

service MathService {
  rpc Add(AddRequest) returns (AddResponse);
}
message AddRequest  { int64 a = 1; int64 b = 2; }
message AddResponse { int64 sum = 1; }
```

```bash
capwrap-gen math.proto   # -> math.capnp, math.capnp.go, math.capwrap.go
```

Each `rpc M(Req) returns (Resp)` is flattened: the fields of the request/response
messages become the method's params/results. Unary methods with scalar/string/
bytes fields are supported today.

### 3. Server — a plain Go type

```go
type calculator struct{}

func (calculator) SayHello(_ context.Context, req *calc.SayHelloRequest) (*calc.SayHelloResponse, error) {
	return &calc.SayHelloResponse{Message: "hello " + req.Name}, nil
}

func (calculator) Add(_ context.Context, req *calc.AddRequest) (*calc.AddResponse, error) {
	return &calc.AddResponse{Sum: req.A + req.B}, nil
}

func main() {
	srv := capwrap.NewServer()
	calc.RegisterCalculatorServer(srv, calculator{})
	log.Fatal(srv.ListenAndServe(context.Background(), "127.0.0.1:7000"))
}
```

### 4. Client — ordinary Go calls

```go
client, err := calc.DialCalculator(ctx, "127.0.0.1:7000")
if err != nil {
	log.Fatal(err)
}
defer client.Close()

hello, _ := client.SayHello(ctx, &calc.SayHelloRequest{Name: "Huy"})
sum, _ := client.Add(ctx, &calc.AddRequest{A: 123, B: 456})
```

## Run the example

The repo ships a working calculator example (bindings already generated):

```bash
go run ./examples/calculator/server
# in another terminal:
go run ./examples/calculator/client -name Huy -a 123 -b 456
```

Output:

```
sayHello -> hello Huy from Cap'n Proto RPC
add(123, 456) -> 579
```

## Project layout

| Path                   | What it is                                       |
| ---------------------- | ------------------------------------------------ |
| `runtime/`             | Small `Dial` / `Serve` runtime (package capwrap) |
| `cmd/capwrap-gen/`     | The wrapper generator CLI                        |
| `internal/generator/`  | Codegen: `.capnp` parser + templates             |
| `internal/protoconv/`  | `.proto` → `.capnp` translation                  |
| `examples/calculator/` | End-to-end schema, server and client             |
| `docs/`                | Design notes and Cap'n Proto vs gRPC             |

## Status & roadmap

Released (`v0.0.1`): single-interface services, unary methods, scalar/`Text`/
`Data` fields, blocking client calls with `context`, `.proto` input, and
one-command codegen (auto `capnp compile`). Methods using lists, nested structs,
unions, or capability parameters generate a "not supported" server stub and are
omitted from the typed client, so the rest of a service still works.

Planned for `v0.1.0`:

- list and nested-struct fields (Cap'n Proto `List` / struct, proto `repeated` /
  message fields),
- async client calls and promise pipelining,
- unary interceptors (logging/metrics/recovery),
- a status/error-code model,
- benchmarks under `benchmarks/`.

## License

[MIT](LICENSE)
