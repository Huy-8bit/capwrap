package generator

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestGenerateGolden(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "calculator.capnp"))
	if err != nil {
		t.Fatal(err)
	}

	got, skipped, err := Generate(string(src))
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join("testdata", "calculator.capwrap.go.golden")
	if *update {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("generated output differs from golden; run `go test ./internal/generator -update`")
	}

	if len(skipped) != 1 || !strings.Contains(skipped[0], "summarize") {
		t.Errorf("expected summarize to be skipped, got %v", skipped)
	}
}

func TestGenerateUnsupportedStub(t *testing.T) {
	// A method using an unsupported type must still produce a server stub so the
	// generated adapter satisfies the capnp interface.
	got, _, err := Generate(string(mustRead(t, "testdata", "calculator.capnp")))
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	if !strings.Contains(out, "func (s calculatorServerAdapter) Summarize(") {
		t.Error("missing Summarize server stub for unsupported method")
	}
	if strings.Contains(out, "func (c *calculatorClient) Summarize(") {
		t.Error("unsupported method should not appear on the client")
	}
}

func TestParseRejectsSchemaWithoutInterface(t *testing.T) {
	if _, _, err := Generate("@0x1234;\n$Go.package(\"x\");\n"); err == nil {
		t.Error("expected error for schema without interface")
	}
}

func mustRead(t *testing.T, parts ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
