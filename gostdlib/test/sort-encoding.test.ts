import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import type { TextMarshaler } from "../src/encoding.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import { Search, Sort, Stable, Strings, type Interface } from "../src/sort.js";

const sortableType = Object.freeze({});

class Sortable extends ProviderInterfaceValue implements Interface {
  constructor(readonly values: { readonly key: number; readonly label: string }[]) {
    super(sortableType);
  }

  Len(): number {
    return this.values.length;
  }

  Less(left: number, right: number): boolean {
    return (this.values[left]?.key ?? 0) < (this.values[right]?.key ?? 0);
  }

  Swap(left: number, right: number): void {
    const saved = this.values[left];
    if (saved === undefined || this.values[right] === undefined) {
      throw new Error("test index is outside the sortable value");
    }
    this.values[left] = this.values[right];
    this.values[right] = saved;
  }
}

test("sort.Search finds the first true index", (): void => {
  assert.equal(Search(10, (index): boolean => index >= 6), 6);
  assert.equal(Search(10, (): boolean => false), 10);
  assert.equal(Search(0, (): boolean => true), 0);
});

test("sort interface operations are in-place and stable when requested", (): void => {
  const ordinary = new Sortable([
    { key: 3, label: "c" },
    { key: 1, label: "a" },
    { key: 2, label: "b" },
  ]);
  Sort(ordinary);
  assert.deepEqual(ordinary.values.map(({ key }): number => key), [1, 2, 3]);

  const stable = new Sortable([
    { key: 2, label: "first" },
    { key: 1, label: "middle" },
    { key: 2, label: "second" },
  ]);
  Stable(stable);
  assert.deepEqual(stable.values.map(({ label }): string => label), [
    "middle",
    "first",
    "second",
  ]);
});

test("sort.Strings mutates the selected runtime slice", (): void => {
  const values = RuntimeSlice.literal(["beta", "alpha", "gamma"]);
  Strings(values);
  assert.deepEqual(
    [values.get(0), values.get(1), values.get(2)],
    ["alpha", "beta", "gamma"],
  );
});

test("encoding.TextMarshaler remains a static contract", (): void => {
  const accepts = (_value: TextMarshaler): void => {};
  assert.equal(typeof accepts, "function");
});
