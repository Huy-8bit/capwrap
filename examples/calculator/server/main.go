// Command server runs the example Calculator service over Cap'n Proto RPC using
// the capwrap runtime. The service is implemented as a plain Go type.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/Huy-8bit/capwrap/examples/calculator/calc"
	capwrap "github.com/Huy-8bit/capwrap/runtime"
)

// calculator implements calc.CalculatorServer with ordinary Go methods.
type calculator struct{}

func (calculator) SayHello(_ context.Context, req *calc.SayHelloRequest) (*calc.SayHelloResponse, error) {
	return &calc.SayHelloResponse{
		Message: fmt.Sprintf("hello %s from Cap'n Proto RPC", req.Name),
	}, nil
}

func (calculator) Add(_ context.Context, req *calc.AddRequest) (*calc.AddResponse, error) {
	return &calc.AddResponse{Sum: req.A + req.B}, nil
}

func main() {
	addr := flag.String("addr", "127.0.0.1:7000", "TCP address to listen on")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := capwrap.NewServer()
	calc.RegisterCalculatorServer(srv, calculator{})

	log.Printf("calculator listening on %s", *addr)
	if err := srv.ListenAndServe(ctx, *addr); err != nil {
		log.Fatal(err)
	}
}
