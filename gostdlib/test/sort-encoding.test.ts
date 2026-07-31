import assert from "node:assert/strict";
import test from "node:test";

import type { TextMarshaler } from "../src/encoding.js";
import { Search } from "../src/sort.js";

test("sort.Search finds the first true index", (): void => {
  assert.equal(Search(10, (index): boolean => index >= 6), 6);
  assert.equal(Search(10, (): boolean => false), 10);
  assert.equal(Search(0, (): boolean => true), 0);
});

test("encoding.TextMarshaler remains a static contract", (): void => {
  const accepts = (_value: TextMarshaler): void => {};
  assert.equal(typeof accepts, "function");
});
