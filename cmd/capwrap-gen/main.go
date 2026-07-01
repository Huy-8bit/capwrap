// Command capwrap-gen generates gRPC-like Go wrappers (*.capwrap.go) from a
// Cap'n Proto schema. The wrapper sits on top of the code produced by the
// normal Cap'n Proto Go compiler (capnpc-go), so run `capnp compile -ogo` first.
//
// It also accepts a protobuf .proto file: it is translated to a sibling .capnp
// (which you then feed to `capnp compile`) and the wrapper is generated from it.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Huy-8bit/capwrap/internal/generator"
	"github.com/Huy-8bit/capwrap/internal/protoconv"
)

func main() {
	out := flag.String("o", "", "output file (default: <schema>.capwrap.go beside the input)")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *out); err != nil {
		fmt.Fprintln(os.Stderr, "capwrap-gen:", err)
		os.Exit(1)
	}
}

func run(schemaPath, outPath string) error {
	switch filepath.Ext(schemaPath) {
	case ".capnp":
	case ".proto":
		var err error
		if schemaPath, err = translateProto(schemaPath); err != nil {
			return err
		}
	default:
		return fmt.Errorf("expected a .capnp or .proto file, got %q", schemaPath)
	}

	src, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}

	code, skipped, err := generator.Generate(string(src))
	if err != nil {
		return err
	}

	if outPath == "" {
		base := strings.TrimSuffix(schemaPath, ".capnp")
		outPath = base + ".capwrap.go"
	}
	if err := os.WriteFile(outPath, code, 0o644); err != nil {
		return err
	}

	fmt.Printf("wrote %s\n", outPath)
	for _, s := range skipped {
		fmt.Fprintf(os.Stderr, "warning: skipped %s (unsupported by the MVP generator)\n", s)
	}
	checkCapnpArtifacts(schemaPath)
	return nil
}

// translateProto converts a .proto file into a sibling .capnp file and returns
// its path. The .capnp is what `capnp compile` and the wrapper generator consume.
func translateProto(protoPath string) (string, error) {
	src, err := os.ReadFile(protoPath)
	if err != nil {
		return "", err
	}
	capnpSrc, skipped, err := protoconv.ToCapnp(string(src))
	if err != nil {
		return "", err
	}
	capnpPath := strings.TrimSuffix(protoPath, ".proto") + ".capnp"
	if err := os.WriteFile(capnpPath, []byte(capnpSrc), 0o644); err != nil {
		return "", err
	}
	fmt.Printf("translated %s -> %s\n", protoPath, capnpPath)
	for _, s := range skipped {
		fmt.Fprintf(os.Stderr, "warning: skipped %s\n", s)
	}
	return capnpPath, nil
}

// checkCapnpArtifacts warns when the companion capnpc-go output is missing,
// since the generated wrapper depends on it to compile.
func checkCapnpArtifacts(schemaPath string) {
	companion := strings.TrimSuffix(schemaPath, ".capnp") + ".capnp.go"
	if _, err := os.Stat(companion); err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "\nnote: %s not found.\n", companion)
	fmt.Fprintln(os.Stderr, "The generated wrapper needs the Cap'n Proto Go bindings. Generate them with:")
	fmt.Fprintf(os.Stderr, "  capnp compile -I <capnp-std> -ogo %s\n", schemaPath)
	if _, err := exec.LookPath("capnp"); err != nil {
		fmt.Fprintln(os.Stderr, "The `capnp` compiler is not installed. See https://capnproto.org/install.html")
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: capwrap-gen [-o output.go] schema.capnp | schema.proto")
	flag.PrintDefaults()
}
