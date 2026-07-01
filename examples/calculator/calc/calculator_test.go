package calc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Huy-8bit/capwrap/examples/calculator/calc"
	capwrap "github.com/Huy-8bit/capwrap/runtime"
)

type calculator struct{}

func (calculator) SayHello(_ context.Context, req *calc.SayHelloRequest) (*calc.SayHelloResponse, error) {
	return &calc.SayHelloResponse{Message: "hello " + req.Name}, nil
}

func (calculator) Add(_ context.Context, req *calc.AddRequest) (*calc.AddResponse, error) {
	return &calc.AddResponse{Sum: req.A + req.B}, nil
}

// TestCalculatorRoundTrip exercises the runtime, generated wrapper and capnp
// bindings together over a real loopback connection.
func TestCalculatorRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := capwrap.NewServer()
	calc.RegisterCalculatorServer(srv, calculator{})
	go srv.Serve(ctx, ln)

	client, err := calc.DialCalculator(ctx, ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	hello, err := client.SayHello(ctx, &calc.SayHelloRequest{Name: "Huy"})
	if err != nil {
		t.Fatal(err)
	}
	if hello.Message != "hello Huy" {
		t.Errorf("SayHello = %q, want %q", hello.Message, "hello Huy")
	}

	sum, err := client.Add(ctx, &calc.AddRequest{A: 123, B: 456})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Sum != 579 {
		t.Errorf("Add = %d, want 579", sum.Sum)
	}
}
