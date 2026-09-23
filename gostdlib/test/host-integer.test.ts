import assert from "node:assert/strict";
import test from "node:test";

import {
  hostInteger,
  integerFromHost,
  unsignedIntegerFromHost,
} from "../src/internal/host-integer.js";

test("host integer boundaries preserve every admitted value", () => {
  assert.equal(hostInteger(9_007_199_254_740_991n), 9_007_199_254_740_991);
  assert.equal(integerFromHost(-9_007_199_254_740_991), -9_007_199_254_740_991n);
  assert.equal(unsignedIntegerFromHost(9_007_199_254_740_991), 9_007_199_254_740_991n);
  for (const value of [9_007_199_254_740_992n, 1n << 63n, 1n << 100n]) {
    assert.equal(hostInteger(value), Number(value));
    assert.equal(integerFromHost(Number(value)), value);
    assert.equal(unsignedIntegerFromHost(Number(value)), value);
    assert.equal(hostInteger(-value), -Number(value));
    assert.equal(integerFromHost(-Number(value)), -value);
  }
});

test("host integer boundaries reject lossy or invalid values", () => {
  assert.throws(() => hostInteger(9_007_199_254_740_993n), RangeError);
  for (const value of [Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY, 1.5]) {
    assert.throws(() => integerFromHost(value), RangeError);
    assert.throws(() => unsignedIntegerFromHost(value), RangeError);
  }
  assert.throws(() => integerFromHost(1.5), RangeError);
  assert.throws(() => unsignedIntegerFromHost(-1), RangeError);
});
