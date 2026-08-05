import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  CmpCompareKernel as Compare,
  CmpOrKernel as Or,
} from "../src/internal/facets/generic-cmp-kernel.js";
import {
  Abs,
  Copysign,
  Float32bits,
  Float32frombits,
  Float64bits,
  Float64frombits,
  Inf,
  IsInf,
  IsNaN,
  Log10,
  MaxFloat64,
  Min,
  Mod,
  NaN,
  Round,
  Signbit,
} from "../src/math.js";
import {
  Add64,
  Div64,
  Len,
  Mul64,
  OnesCount,
  OnesCount32,
  OnesCount64,
  ReverseBytes32,
  ReverseBytes64,
  RotateLeft32,
  RotateLeft64,
} from "../src/math/bits.js";
import { Uint64 } from "../src/math/rand/v2.js";

test("cmp.Compare follows Go ordering including NaN", (): void => {
  const less = <T extends number | string>(left: T, right: T): boolean => left < right;
  const equal = <T extends number | string>(left: T, right: T): boolean => left === right;
  assert.equal(Compare(less, equal, 3, 8), -1n);
  assert.equal(Compare(less, equal, "z", "a"), 1n);
  assert.equal(Compare(less, equal, Number.NaN, 0), -1n);
  assert.equal(Compare(less, equal, Number.NaN, Number.NaN), 0n);
  assert.equal(Compare(less, equal, -0, 0), 0n);
});

test("cmp.Or returns the first non-zero copied value", (): void => {
  const values = RuntimeSlice.literal([0, 0, 7, 9]);
  assert.equal(Or(
    (value): number => value,
    (left, right): boolean => left === right,
    (value): number => value,
    (): number => 0,
    values,
  ), 7);
  assert.equal(Or(
    (value): string => value,
    (left, right): boolean => left === right,
    (value): string => value,
    (): string => "",
    RuntimeSlice.literal(["", "selected", "later"]),
  ), "selected");
});

test("math preserves selected scalar semantics", (): void => {
  assert.equal(Abs(-7.5), 7.5);
  assert.equal(Object.is(Copysign(4, -0), -4), true);
  assert.equal(Float32bits(1), 0x3f80_0000);
  assert.equal(Float32frombits(0x3f80_0000), 1);
  assert.equal(Object.is(Float32frombits(0x8000_0000), -0), true);
  assert.equal(Float64bits(1), 4_607_182_418_800_017_408n);
  assert.equal(Float64frombits(4_607_182_418_800_017_408n), 1);
  assert.equal(Object.is(Float64frombits(9_223_372_036_854_775_808n), -0), true);
  assert.equal(IsInf(Inf(1n), 1n), true);
  assert.equal(IsInf(Inf(-1n), -1n), true);
  assert.equal(IsNaN(NaN()), true);
  assert.equal(Log10(1_000), 3);
  assert.equal(Round(1.5), 2);
  assert.equal(Round(-1.5), -2);
  assert.equal(Object.is(Round(-0.1), -0), true);
  assert.equal(Signbit(-0), true);
  assert.equal(Signbit(0), false);
  assert.equal(Object.is(Min(0, -0), -0), true);
  assert.equal(Mod(-7, 4), -3);
  assert.equal(MaxFloat64, Number.MAX_VALUE);
});

test("math/bits counts the selected widths", (): void => {
  assert.equal(OnesCount(0x0fff_ffff_ffffn), 44n);
  assert.equal(OnesCount32(0xf0f0_000f), 12n);
});

test("math/bits splits represented 128-bit products", (): void => {
  assert.deepEqual(Mul64(0x1_0000_0000n, 0x1_0000_0000n), [1n, 0n]);
  assert.deepEqual(Mul64(0x1_0000_0000n, 0x1_0000n), [0n, 0x1_0000_0000_0000n]);
});

test("math/bits preserves the reached TS-Go operation surface", (): void => {
  assert.deepEqual(Add64(0xffff_ffff_ffff_f000n, 0x0fffn, 1n), [0n, 1n]);
  assert.deepEqual(Div64(1n, 0n, 2n), [0x8000_0000_0000_0000n, 0n]);
  assert.equal(Len(0x1_0000_0000n), 33n);
  assert.equal(OnesCount64(0xf000_0000_000fn), 8n);
  assert.equal(ReverseBytes32(0x0102_0304), 0x0403_0201);
  assert.equal(ReverseBytes64(0x0102_0304_0506_0000n), 0x0000_0605_0403_0201n);
  assert.equal(RotateLeft32(0x1234_5678, 8n), 0x3456_7812);
  assert.equal(RotateLeft32(0x1234_5678, -8n), 0x7812_3456);
  assert.equal(RotateLeft64(0x0000_1234_5678_0000n, 16n), 0x1234_5678_0000_0000n);
  assert.equal(RotateLeft64(0x0000_1234_5678_0000n, -16n), 0x0000_0000_1234_5678n);
});

test("math/bits Div64 preserves the Go overflow boundary", (): void => {
  assert.throws((): void => {
    Div64(2n, 0n, 2n);
  });
});

test("math/rand/v2 returns a represented uint64", (): void => {
  for (let sample = 0; sample < 32; sample += 1) {
    const value = Uint64();
    assert.equal(typeof value, "bigint");
    assert.equal(value >= 0n, true);
    assert.equal(value <= 18_446_744_073_709_551_615n, true);
  }
});
