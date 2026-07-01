@0xc7d2d2b67fd7ab31;

using Go = import "/go.capnp";

$Go.package("calc");
$Go.import("github.com/Huy-8bit/capwrap/examples/calculator/calc");

interface Calculator {
  sayHello @0 (name :Text) -> (message :Text);
  add @1 (a :Int64, b :Int64) -> (sum :Int64);

  # summarize is intentionally left beyond the MVP wrapper: it uses List(struct)
  # parameters, so capwrap-gen emits a "not supported" server stub for it and no
  # typed Go client method. It shows how unsupported methods degrade gracefully.
  summarize @2 (items :List(ComplexItem)) -> (
    count :UInt64,
    checksum :UInt64,
    amountSum :Int64,
    scoreSum :Float64,
  );
}

struct ComplexItem {
  id @0 :UInt64;
  category @1 :UInt32;
  score @2 :Float64;
  amount @3 :Int64;
  flag @4 :Bool;
  label @5 :Text;
  auxA @6 :UInt64;
  auxB @7 :UInt64;
  ratio @8 :Float64;
}
