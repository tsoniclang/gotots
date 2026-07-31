import assert from "node:assert/strict";
import test from "node:test";

import { GoMap } from "@gotots/runtime/map.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Seq } from "../src/iter.js";
import {
  KeysCooperative,
} from "../src/internal/portable/maps/cooperative.js";
import {
  CollectCooperative,
  ContainsFuncCooperative,
  SortStableFuncCooperative,
} from "../src/internal/portable/slices/cooperative.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import {
  Copy,
  EqualFunc as MapsEqualFunc,
  Keys,
  Values as MapValues,
} from "../src/maps.js";
import {
  AppendSeq,
  BinarySearch,
  BinarySearchFunc,
  Clip,
  Clone,
  Collect,
  Compact,
  CompactFunc,
  Compare,
  CompareFunc,
  Concat,
  Contains,
  ContainsFunc,
  Delete,
  DeleteFunc,
  Equal,
  EqualFunc,
  Index,
  IndexFunc,
  Insert,
  Repeat,
  Replace,
  Reverse,
  Sort,
  SortFunc,
  SortStableFunc,
  Sorted,
  SortedFunc,
  Values,
} from "../src/slices.js";

function values<T>(source: RuntimeSlice<T>): T[] {
  return sliceValues(source);
}

test("slices search and comparison operations are generic", (): void => {
  const source = RuntimeSlice.literal([1, 3, 3, 8]);
  assert.deepEqual(BinarySearch(source, 3), [1, true]);
  assert.deepEqual(BinarySearch(source, 4), [3, false]);
  assert.deepEqual(BinarySearchFunc(source, "3", (value, target): number => value - Number(target)), [1, true]);
  assert.equal(Contains(source, 8), true);
  assert.equal(ContainsFunc(source, (value): boolean => value > 5), true);
  assert.equal(Index(source, 3), 1);
  assert.equal(IndexFunc(source, (value): boolean => value % 2 === 0), 3);
  assert.equal(Equal(source, RuntimeSlice.literal([1, 3, 3, 8])), true);
  assert.equal(EqualFunc(source, RuntimeSlice.literal(["1", "3", "3", "8"]), (left, right): boolean => String(left) === right), true);
  assert.equal(Compare(RuntimeSlice.literal(["a", "b"]), RuntimeSlice.literal(["a", "c"])), -1);
  assert.equal(CompareFunc(source, RuntimeSlice.literal([1, 3, 4]), (left, right): number => left - right), -1);
});

test("slices transformations preserve order and nilness", (): void => {
  const source = RuntimeSlice.literal([1, 1, 2, 2, 3]);
  assert.deepEqual(values(Compact(source)), [1, 2, 3]);
  assert.deepEqual(values(CompactFunc(RuntimeSlice.literal(["A", "a", "B"]), (left, right): boolean => left.toLowerCase() === right.toLowerCase())), ["A", "B"]);
  assert.deepEqual(values(Delete(RuntimeSlice.literal([0, 1, 2, 3]), 1, 3)), [0, 3]);
  assert.deepEqual(values(DeleteFunc(RuntimeSlice.literal([1, 2, 3, 4]), (value): boolean => value % 2 === 0)), [1, 3]);
  assert.deepEqual(values(Insert(RuntimeSlice.literal([1, 4]), 1, RuntimeSlice.literal([2, 3]))), [1, 2, 3, 4]);
  assert.deepEqual(values(Repeat(RuntimeSlice.literal(["x", "y"]), 3)), ["x", "y", "x", "y", "x", "y"]);
  assert.deepEqual(values(Replace(RuntimeSlice.literal([1, 2, 3, 4]), 1, 3, RuntimeSlice.literal([8, 9]))), [1, 8, 9, 4]);
  assert.equal(Clone(RuntimeSlice.nil<number>()).isNil(), true);
  assert.equal(Clip(RuntimeSlice.literal([1, 2])).capacity, 2);
});

test("slices sequence and sorting operations use one typed path", (): void => {
  const sequence = new Seq<number>((yieldValue): void => {
    yieldValue?.(3);
    yieldValue?.(1);
    yieldValue?.(2);
  });
  assert.deepEqual(values(Collect(sequence)), [3, 1, 2]);
  assert.deepEqual(values(AppendSeq(RuntimeSlice.literal([0]), sequence)), [0, 3, 1, 2]);
  assert.deepEqual(values(Concat(RuntimeSlice.literal([
    RuntimeSlice.literal([1, 2]),
    RuntimeSlice.literal([3]),
  ]))), [1, 2, 3]);

  const ordered = RuntimeSlice.literal([3, Number.NaN, 1, 2]);
  Sort(ordered);
  assert.equal(Number.isNaN(ordered.get(0)), true);
  assert.deepEqual(sliceValues(ordered).slice(1), [1, 2, 3]);

  const descending = RuntimeSlice.literal([1, 3, 2]);
  SortFunc(descending, (left, right): number => right - left);
  assert.deepEqual(values(descending), [3, 2, 1]);

  const stable = RuntimeSlice.literal([
    { key: 1, order: "a" },
    { key: 0, order: "b" },
    { key: 1, order: "c" },
  ]);
  SortStableFunc(stable, (left, right): number => left.key - right.key);
  assert.deepEqual(values(stable).map((entry): string => entry.order), ["b", "a", "c"]);
  assert.deepEqual(values(Sorted(sequence)), [1, 2, 3]);
  assert.deepEqual(values(SortedFunc(sequence, (left, right): number => right - left)), [3, 2, 1]);

  const yielded: number[] = [];
  Values(RuntimeSlice.literal([4, 5])).value?.((value): boolean => {
    yielded.push(value);
    return true;
  });
  assert.deepEqual(yielded, [4, 5]);
  const reversed = RuntimeSlice.literal([1, 2, 3]);
  Reverse(reversed);
  assert.deepEqual(values(reversed), [3, 2, 1]);
});

test("maps functions use the semantic GoMapValue contract", (): void => {
  const first = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  const second = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  assert.equal(MapsEqualFunc(first, second, (left, right): boolean => left === right), true);
  const target = GoMap.make<string, number>(0, 0, []);
  Copy(target, first);
  assert.deepEqual(target.keys().sort(), ["a", "b"]);

  const keys: string[] = [];
  Keys(first).value?.((key): boolean => {
    keys.push(key);
    return true;
  });
  assert.deepEqual(keys.sort(), ["a", "b"]);

  const mapValues: number[] = [];
  MapValues(first).value?.((value): boolean => {
    mapValues.push(value);
    return true;
  });
  assert.deepEqual(mapValues.sort(), [1, 2]);
});

test("cooperative collection facets have one typed internal owner", async (): Promise<void> => {
  const sequence = new Seq<
    number,
    ((yieldValue: ((value: number) => Promise<boolean>) | undefined) => Promise<void>) | undefined
  >(async (yieldValue): Promise<void> => {
    if (yieldValue !== undefined) {
      await yieldValue(3);
      await yieldValue(1);
      await yieldValue(2);
    }
  });
  assert.deepEqual(values(await CollectCooperative(sequence)), [3, 1, 2]);
  assert.equal(
    await ContainsFuncCooperative(
      RuntimeSlice.literal([1, 2, 3]),
      async (value): Promise<boolean> => value === 2,
    ),
    true,
  );
  const sortable = RuntimeSlice.literal([
    { group: 1, order: "a" },
    { group: 0, order: "b" },
    { group: 1, order: "c" },
  ]);
  await SortStableFuncCooperative(
    sortable,
    async (left, right): Promise<number> => left.group - right.group,
  );
  assert.deepEqual(
    sliceValues(sortable).map((entry): string => entry.order),
    ["b", "a", "c"],
  );

  const source = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  const keys: string[] = [];
  await KeysCooperative(source).value?.(async (key): Promise<boolean> => {
    keys.push(key);
    return true;
  });
  assert.deepEqual(keys.sort(), ["a", "b"]);
});
