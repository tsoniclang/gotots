import assert from "node:assert/strict";
import test from "node:test";

import { Compare } from "../src/cmp.js";
import {
  Abs,
  Copysign,
  Float64bits,
  Inf,
  IsInf,
  IsNaN,
  MaxFloat64,
  Min,
  Mod,
  NaN,
} from "../src/math.js";
import { OnesCount, OnesCount32 } from "../src/math/bits.js";
import { Uint64 } from "../src/math/rand/v2.js";

test("cmp.Compare follows Go ordering including NaN", (): void => {
  assert.equal(Compare(3, 8), -1);
  assert.equal(Compare("z", "a"), 1);
  assert.equal(Compare(Number.NaN, 0), -1);
  assert.equal(Compare(Number.NaN, Number.NaN), 0);
  assert.equal(Compare(-0, 0), 0);
});

test("math preserves selected scalar semantics", (): void => {
  assert.equal(Abs(-7.5), 7.5);
  assert.equal(Object.is(Copysign(4, -0), -4), true);
  assert.equal(Float64bits(1), 4_607_182_418_800_017_400);
  assert.equal(IsInf(Inf(1), 1), true);
  assert.equal(IsInf(Inf(-1), -1), true);
  assert.equal(IsNaN(NaN()), true);
  assert.equal(Object.is(Min(0, -0), -0), true);
  assert.equal(Mod(-7, 4), -3);
  assert.equal(MaxFloat64, Number.MAX_VALUE);
});

test("math/bits counts the selected widths", (): void => {
  assert.equal(OnesCount(0x0fff_ffff_ffff), 44);
  assert.equal(OnesCount32(0xf0f0_000f), 12);
});

test("math/rand/v2 returns a represented uint64", (): void => {
  for (let sample = 0; sample < 32; sample += 1) {
    const value = Uint64();
    assert.equal(Number.isInteger(value), true);
    assert.equal(value >= 0, true);
    assert.equal(value <= 18_446_744_073_709_551_615, true);
  }
});
