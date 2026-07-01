package protoconv

import (
	"strings"
	"testing"
)

const sampleProto = `syntax = "proto3";
package math;
option go_package = "capwrapdemo/mathpb;mathpb";

service MathService {
  rpc Add(AddRequest) returns (AddResponse);
  rpc Ping(stream PingRequest) returns (PingResponse); // streaming -> skipped
}

message AddRequest { int64 a = 1; int64 b = 2; }
message AddResponse { int64 sum = 1; }
message PingRequest { string msg = 1; }
message PingResponse { bool ok = 1; }
`

func TestToCapnp(t *testing.T) {
	out, skipped, err := ToCapnp(sampleProto)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`$Go.package("mathpb");`,
		`$Go.import("capwrapdemo/mathpb");`,
		`interface MathService {`,
		`add @0 (a :Int64, b :Int64) -> (sum :Int64);`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}

	if len(skipped) != 1 || !strings.Contains(skipped[0], "Ping") {
		t.Errorf("expected streaming Ping to be skipped, got %v", skipped)
	}
	if strings.Contains(out, "ping @") {
		t.Error("streaming method should not appear in the interface")
	}
}

func TestSnakeCaseFieldsBecomeCamel(t *testing.T) {
	out, _, err := ToCapnp(`option go_package = "x/y;y";
service S { rpc M(Req) returns (Resp); }
message Req { string first_name = 1; }
message Resp { int32 char_count = 1; }
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "firstName :Text") || !strings.Contains(out, "charCount :Int32") {
		t.Errorf("snake_case not converted to camelCase:\n%s", out)
	}
}

func TestMissingGoPackageErrors(t *testing.T) {
	_, _, err := ToCapnp("service S { rpc M(Req) returns (Resp); }\nmessage Req {}\nmessage Resp {}\n")
	if err == nil {
		t.Error("expected error when go_package is missing")
	}
}
