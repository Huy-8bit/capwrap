// Command client calls the example Calculator service. Every call is an
// ordinary Go method call taking a context and a request struct.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/Huy-8bit/capwrap/examples/calculator/calc"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7000", "calculator service address")
	name := flag.String("name", "world", "name to greet")
	a := flag.Int64("a", 40, "first add operand")
	b := flag.Int64("b", 2, "second add operand")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := calc.DialCalculator(ctx, *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	hello, err := client.SayHello(ctx, &calc.SayHelloRequest{Name: *name})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sayHello -> %s", hello.Message)

	sum, err := client.Add(ctx, &calc.AddRequest{A: *a, B: *b})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("add(%d, %d) -> %d", *a, *b, sum.Sum)
}
