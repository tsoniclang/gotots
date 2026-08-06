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
});

test("host integer boundaries reject lossy or invalid values", () => {
  assert.throws(() => hostInteger(9_007_199_254_740_992n), RangeError);
  assert.throws(() => integerFromHost(1.5), RangeError);
  assert.throws(() => unsignedIntegerFromHost(-1), RangeError);
});
