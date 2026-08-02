import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import {
  SlicesGrowKernel as Grow,
} from "../src/internal/facets/generic-slices-kernel.js";

interface Cell {
  value: number;
}

const copyCell = (value: Cell): Cell => ({ value: value.value });
const zeroCell = (): Cell => ({ value: 0 });
const sliceValue = <T>(value: RuntimeSlice<T>): RuntimeSlice<T> => value;
const storageValue = <T>(value: T): T => value;

test("slices.Grow preserves Go length, capacity, alias, copy, and zero semantics", () => {
  const source = RuntimeSlice.make<Cell>(1, 1, zeroCell());
  source.set(0, { value: 7 });
  const grown = Grow(
    sliceValue,
    sliceValue,
    copyCell,
    storageValue,
    storageValue,
    zeroCell,
    source,
    3,
  );
  assert.equal(grown.length, 1);
  assert.ok(grown.capacity >= 4);
  grown.get(0).value = 9;
  assert.equal(source.get(0).value, 7);

  const exposed = grown.slice(0, grown.capacity, null);
  exposed.get(1).value = 5;
  assert.equal(exposed.get(2).value, 0);

  assert.equal(Grow(
    sliceValue,
    sliceValue,
    copyCell,
    storageValue,
    storageValue,
    zeroCell,
    grown,
    0,
  ), grown);
  const nil = RuntimeSlice.nil<Cell>();
  assert.equal(Grow(
    sliceValue,
    sliceValue,
    copyCell,
    storageValue,
    storageValue,
    zeroCell,
    nil,
    0,
  ), nil);
  assert.throws(
    () => Grow(
      sliceValue,
      sliceValue,
      copyCell,
      storageValue,
      storageValue,
      zeroCell,
      nil,
      -1,
    ),
    (failure: unknown): boolean => failure instanceof GoPanic
      && failure.value.$go$format("v", "", undefined) === "cannot be negative",
  );
});
