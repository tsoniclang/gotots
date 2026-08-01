import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  Clone,
  Compare,
  Cut,
  Equal,
  IndexAny,
  IndexByte,
  Join,
  LastIndexByte,
  Trim,
  TrimLeft,
  TrimRight,
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

test("bytes search, clone, join, and trim preserve byte boundaries", () => {
  const source = RuntimeSlice.literal([0x61, 0xc3, 0xa9, 0x62, 0x61]);
  const clone = Clone(source);
  assert.deepEqual(sliceValues(clone), sliceValues(source));
  clone.set(0, 0x78);
  assert.equal(source.get(0), 0x61);
  assert.equal(Compare(source, RuntimeSlice.literal([0x61, 0xc3, 0xa9, 0x63])), -1);
  assert.equal(IndexAny(source, "Ã©"), 1);
  assert.equal(IndexByte(source, 0x62), 3);
  assert.equal(LastIndexByte(source, 0x61), 4);
  assert.deepEqual(
    sliceValues(Join(RuntimeSlice.literal([
      RuntimeSlice.literal([1]),
      RuntimeSlice.literal([2, 3]),
    ]), RuntimeSlice.literal([9]))),
    [1, 9, 2, 3],
  );
  assert.deepEqual(sliceValues(Trim(RuntimeSlice.literal([1, 2, 3, 2]), "\x01\x02")), [3]);
  assert.deepEqual(sliceValues(TrimLeft(RuntimeSlice.literal([1, 2, 3]), "\x01\x02")), [3]);
  assert.deepEqual(sliceValues(TrimRight(RuntimeSlice.literal([3, 1, 2]), "\x01\x02")), [3]);
});
