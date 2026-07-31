import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  Cut,
  Equal,
  TrimSpace,
} from "../src/bytes.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

test("bytes comparison and cutting retain Go slice identity", () => {
  const source = RuntimeSlice.literal([1, 2, 3, 2, 3]);
  const separator = RuntimeSlice.literal([2, 3]);
  const [before, after, found] = Cut(source, separator);
  assert.equal(found, true);
  assert.deepEqual(sliceValues(before), [1]);
  assert.deepEqual(sliceValues(after), [2, 3]);
  after.set(0, 9);
  assert.equal(source.get(3), 9);
  assert.equal(Equal(source, RuntimeSlice.literal([1, 2, 3, 9, 3])), true);
});

test("bytes TrimSpace uses Go Unicode whitespace over UTF-8 bytes", () => {
  const source = RuntimeSlice.literal([
    0xe3, 0x80, 0x80,
    0x41,
    0xc2, 0xa0,
  ]);
  const trimmed = TrimSpace(source);
  assert.deepEqual(sliceValues(trimmed), [0x41]);
});
