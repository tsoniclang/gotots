import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  readElement,
  storeElement,
} from "../src/internal/portable/slices/capabilities.js";

const identity = <T>(value: T): T => value;

test("portable slice loops retain host-number indexes", (): void => {
  const values = RuntimeSlice.literal<number | undefined>([undefined, 2]);

  assert.equal(readElement(values, 0, identity, identity), undefined);
  storeElement(values, 1, 7, identity, identity);
  assert.equal(readElement(values, 1, identity, identity), 7);
  assert.throws(() => readElement(values, 2, identity, identity));
});
