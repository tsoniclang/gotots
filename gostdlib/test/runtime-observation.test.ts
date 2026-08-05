import assert from "node:assert/strict";
import test from "node:test";
import { gunzipSync } from "node:zlib";
import {
  GoInterfaceValue,
  type GoError,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "../src/internal/scalars.js";
import { integerFromHost } from "../src/internal/host-integer.js";
import {
  Caller,
  GC,
  GOMAXPROCS,
  GOARCH,
  GOOS,
  MemStats,
  ReadMemStats,
} from "../src/runtime.js";
import { SetMaxStack, Stack } from "../src/runtime/debug.js";
import {
  All,
  KindFloat64,
  KindFloat64Histogram,
  KindUint64,
  Read,
  Sample,
  Value,
} from "../src/runtime/metrics.js";
import {
  Lookup,
  Profile,
  StartCPUProfile,
  StopCPUProfile,
} from "../src/runtime/pprof.js";
import { Testing } from "../src/testing.js";

class BufferWriter extends GoInterfaceValue {
	static readonly comparable = true;
	readonly $go$type = BufferWriter;
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  readonly $go$formatString = false;
  readonly bytes: number[] = [];

  $go$implements(contract: readonly object[]): boolean {
    return contract.length === 0;
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  $go$format(): string {
    return "buffer writer";
  }

  Write(buffer: RuntimeSlice<uint8>): [int, GoError | undefined] {
    for (let index = 0; index < buffer.length; index += 1) {
      this.bytes.push(buffer.get(index));
    }
    return [integerFromHost(buffer.length), undefined];
  }
}

test("runtime process observations are populated", () => {
  assert.notEqual(GOARCH, "");
  assert.notEqual(GOOS, "");
  assert.equal(GOMAXPROCS(0n), 1n);
  GC();

  const stats = new MemStats();
  ReadMemStats(stats);
  assert.ok(stats.Sys >= stats.HeapAlloc);

  const [, file, line, ok] = Caller(0n);
  assert.equal(ok, true);
  assert.match(file, /runtime-observation/u);
  assert.ok(line > 0n);

  const stack = Stack();
  assert.ok(stack.length > 0);
  const previous = SetMaxStack(2_000_000n);
  assert.ok(previous > 0n);
});

test("runtime profiles write concrete provider observations", async () => {
  const writer = new BufferWriter();
  assert.equal(StartCPUProfile(writer), undefined);
  const duplicate = StartCPUProfile(writer);
  assert.notEqual(duplicate, undefined);
  assert.match(duplicate?.Error() ?? "", /already in use/u);
  await StopCPUProfile();
  assert.ok(writer.bytes.length > 0);
  assert.deepEqual(writer.bytes.slice(0, 2), [0x1f, 0x8b]);
  assert.ok(gunzipSync(Uint8Array.from(writer.bytes)).length > 0);

  const heap = Lookup("heap");
  assert.ok(heap instanceof Profile);
  assert.equal(Profile.WriteTo(heap, writer, 0n), undefined);
  assert.equal(Lookup("missing"), undefined);
});

test("runtime metrics publish typed descriptions and selected values", () => {
  const descriptions = All();
  assert.ok(descriptions.length > 0);

  const samples = RuntimeSlice.literal([
    new Sample("/memory/classes/total:bytes"),
    new Sample("/cpu/classes/user:cpu-seconds"),
  ]);
  Read(samples);

  assert.equal(samples.get(0).Value.Kind().value, KindUint64.value);
  assert.ok(samples.get(0).Value.Uint64() > 0n);
  assert.equal(samples.get(1).Value.Kind().value, KindFloat64.value);
  assert.ok(samples.get(1).Value.Float64() >= 0);
  assert.equal(KindFloat64Histogram.value, 3n);
});

test("Testing reports the active node test runner", () => {
  assert.equal(Testing(), true);
});
