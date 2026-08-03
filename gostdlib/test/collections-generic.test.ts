import assert from "node:assert/strict";
import test from "node:test";

import { GoMap, type GoMapValue } from "@gotots/runtime/map.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Seq } from "../src/iter.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import {
  MapsCopyKernel as Copy,
  MapsEqualFuncKernel as MapsEqualFunc,
  MapsKeysKernel as Keys,
  MapsValuesKernel as MapValues,
} from "../src/internal/facets/generic-maps-kernel.js";
import {
  SlicesAppendSeqKernel as AppendSeq,
  SlicesBinarySearchFuncKernel as BinarySearchFunc,
  SlicesBinarySearchKernel as BinarySearch,
  SlicesClipKernel as Clip,
  SlicesCloneKernel as Clone,
  SlicesCollectKernel as Collect,
  SlicesCompactFuncKernel as CompactFunc,
  SlicesCompactKernel as Compact,
  SlicesCompareFuncKernel as CompareFunc,
  SlicesCompareKernel as Compare,
  SlicesConcatKernel as Concat,
  SlicesContainsFuncKernel as ContainsFunc,
  SlicesContainsKernel as Contains,
  SlicesDeleteFuncKernel as DeleteFunc,
  SlicesDeleteKernel as Delete,
  SlicesEqualFuncKernel as EqualFunc,
  SlicesEqualKernel as Equal,
  SlicesIndexFuncKernel as IndexFunc,
  SlicesIndexKernel as Index,
  SlicesInsertKernel as Insert,
  SlicesRepeatKernel as Repeat,
  SlicesReplaceKernel as Replace,
  SlicesReverseKernel as Reverse,
  SlicesSortFuncKernel as SortFunc,
  SlicesSortKernel as Sort,
  SlicesSortStableFuncKernel as SortStableFunc,
  SlicesSortedFuncKernel as SortedFunc,
  SlicesSortedKernel as Sorted,
  SlicesValuesKernel as Values,
} from "../src/internal/facets/generic-slices-kernel.js";

function values<T>(source: RuntimeSlice<T>): T[] {
  return sliceValues(source);
}

const copyValue = <T>(value: T): T => value;
const equalValue = <T>(left: T, right: T): boolean => left === right;
const lessValue = <T extends number | string>(left: T, right: T): boolean => (
  left < right
);
const sliceValue = <T>(source: RuntimeSlice<T>): RuntimeSlice<T> => source;
const zeroNumber = (): number => 0;

test("slices search and comparison operations are generic", async (): Promise<void> => {
  const source = RuntimeSlice.literal([1, 3, 3, 8]);
  assert.deepEqual(BinarySearch(
    lessValue,
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    source,
    3,
  ), [1, true]);
  assert.deepEqual(BinarySearch(
    lessValue,
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    source,
    4,
  ), [3, false]);
  assert.deepEqual(await BinarySearchFunc(
    sliceValue,
    copyValue,
    copyValue,
    source,
    "3",
    (value: number, target: string): number => value - Number(target),
  ), [1, true]);
  assert.equal(Contains(
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    source,
    8,
  ), true);
  assert.equal(await ContainsFunc(
    sliceValue,
    copyValue,
    copyValue,
    source,
    (value: number): boolean => value > 5,
  ), true);
  assert.equal(Index(
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    source,
    3,
  ), 1);
  assert.equal(await IndexFunc(
    sliceValue,
    copyValue,
    copyValue,
    source,
    (value: number): boolean => value % 2 === 0,
  ), 3);
  assert.equal(Equal(
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    source,
    RuntimeSlice.literal([1, 3, 3, 8]),
  ), true);
  assert.equal(await EqualFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    copyValue,
    source,
    RuntimeSlice.literal(["1", "3", "3", "8"]),
    (left: number, right: string): boolean => String(left) === right,
  ), true);
  assert.equal(Compare<RuntimeSlice<string>, string, string>(
    lessValue,
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    RuntimeSlice.literal(["a", "b"]),
    RuntimeSlice.literal(["a", "c"]),
  ), -1);
  assert.equal(await CompareFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    copyValue,
    source,
    RuntimeSlice.literal([1, 3, 4]),
    (left: number, right: number): number => left - right,
  ), -1);
});

test("slices transformations preserve order and nilness", async (): Promise<void> => {
  const source = RuntimeSlice.literal([1, 1, 2, 2, 3]);
  assert.deepEqual(values(Compact(source)), [1, 2, 3]);
  assert.deepEqual(values(await CompactFunc(
    RuntimeSlice.literal(["A", "a", "B"]),
    (left, right): boolean => left.toLowerCase() === right.toLowerCase(),
  )), ["A", "B"]);
  assert.deepEqual(values(Delete(RuntimeSlice.literal([0, 1, 2, 3]), 1, 3)), [0, 3]);
  assert.deepEqual(values(await DeleteFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroNumber,
    RuntimeSlice.literal([1, 2, 3, 4]),
    (value): boolean => value % 2 === 0,
  )), [1, 3]);
  assert.deepEqual(values(Insert(RuntimeSlice.literal([1, 4]), 1, RuntimeSlice.literal([2, 3]))), [1, 2, 3, 4]);
  assert.deepEqual(values(Repeat(RuntimeSlice.literal(["x", "y"]), 3)), ["x", "y", "x", "y", "x", "y"]);
  assert.deepEqual(values(Replace(RuntimeSlice.literal([1, 2, 3, 4]), 1, 3, RuntimeSlice.literal([8, 9]))), [1, 8, 9, 4]);
  assert.equal(Clone(RuntimeSlice.nil<number>()).isNil(), true);
  assert.equal(Clip(RuntimeSlice.literal([1, 2])).capacity, 2);
});

test("slices sequence and sorting operations use one typed path", async (): Promise<void> => {
  const sequence = new Seq<number>((yieldValue): void => {
    yieldValue?.(3);
    yieldValue?.(1);
    yieldValue?.(2);
  });
  assert.deepEqual(
    values(await Collect(copyValue, copyValue, sequence)),
    [3, 1, 2],
  );
  assert.deepEqual(values(await AppendSeq(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroNumber,
    RuntimeSlice.literal([0]),
    sequence,
  )), [0, 3, 1, 2]);
  assert.deepEqual(values(Concat(RuntimeSlice.literal([
    RuntimeSlice.literal([1, 2]),
    RuntimeSlice.literal([3]),
  ]))), [1, 2, 3]);

  const ordered = RuntimeSlice.literal([3, Number.NaN, 1, 2]);
  Sort<RuntimeSlice<number>, number, number>(
    lessValue,
    sliceValue,
    copyValue,
    equalValue,
    copyValue,
    copyValue,
    ordered,
  );
  assert.equal(Number.isNaN(ordered.get(0)), true);
  assert.deepEqual(sliceValues(ordered).slice(1), [1, 2, 3]);

  const descending = RuntimeSlice.literal([1, 3, 2]);
  await SortFunc(
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    descending,
    (left: number, right: number): number => right - left,
  );
  assert.deepEqual(values(descending), [3, 2, 1]);

  const stable = RuntimeSlice.literal([
    { key: 1, order: "a" },
    { key: 0, order: "b" },
    { key: 1, order: "c" },
  ]);
  await SortStableFunc<
    RuntimeSlice<{ key: number; order: string }>,
    { key: number; order: string },
    { key: number; order: string }
  >(
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    stable,
    (left, right): number => left.key - right.key,
  );
  assert.deepEqual(values(stable).map((entry): string => entry.order), ["b", "a", "c"]);
  assert.deepEqual(values(await Sorted<number, number>(
    lessValue,
    copyValue,
    equalValue,
    copyValue,
    copyValue,
    sequence,
  )), [1, 2, 3]);
  assert.deepEqual(values(await SortedFunc<number, number>(
    copyValue,
    copyValue,
    copyValue,
    sequence,
    (left: number, right: number): number => right - left,
  )), [3, 2, 1]);

  const yielded: number[] = [];
  await Values<RuntimeSlice<number>, number, number>(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([4, 5]),
  ).value?.((value): boolean => {
    yielded.push(value);
    return true;
  });
  assert.deepEqual(yielded, [4, 5]);
  const reversed = RuntimeSlice.literal([1, 2, 3]);
  Reverse(reversed);
  assert.deepEqual(values(reversed), [3, 2, 1]);
});

test("maps functions use the semantic GoMapValue contract", async (): Promise<void> => {
  const mapValue = <K, V>(value: GoMapValue<K, V>): GoMapValue<K, V> => value;
  const first = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  const second = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  assert.equal(
    await MapsEqualFunc(
      mapValue,
      mapValue,
      copyValue,
      copyValue,
      first,
      second,
      (left, right): boolean => left === right,
    ),
    true,
  );
  const target = GoMap.make<string, number>(0, 0, []);
  Copy(mapValue, mapValue, copyValue, copyValue, target, first);
  assert.deepEqual(target.keys().sort(), ["a", "b"]);

  const keys: string[] = [];
  await Keys(mapValue, copyValue, first).value?.((key): boolean => {
    keys.push(key);
    return true;
  });
  assert.deepEqual(keys.sort(), ["a", "b"]);

  const mapValues: number[] = [];
  await MapValues(mapValue, copyValue, first).value?.((value): boolean => {
    mapValues.push(value);
    return true;
  });
  assert.deepEqual(mapValues.sort(), [1, 2]);
});

test("cooperative collection facets have one typed internal owner", async (): Promise<void> => {
  const sequence = new Seq<number>(async (yieldValue): Promise<void> => {
    if (yieldValue !== undefined) {
      await yieldValue(3);
      await yieldValue(1);
      await yieldValue(2);
    }
  });
  assert.deepEqual(values(await Collect(
    copyValue,
    copyValue,
    sequence,
  )), [3, 1, 2]);
  assert.equal(
    await ContainsFunc(
      sliceValue,
      copyValue,
      copyValue,
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
  await SortStableFunc<
    RuntimeSlice<{ group: number; order: string }>,
    { group: number; order: string },
    { group: number; order: string }
  >(
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    sortable,
    async (left, right): Promise<number> => left.group - right.group,
  );
  assert.deepEqual(
    sliceValues(sortable).map((entry): string => entry.order),
    ["b", "a", "c"],
  );

  const source = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  const keys: string[] = [];
  await Keys(
    (value): GoMapValue<string, number> => value,
    (key): string => key,
    source,
  ).value?.(async (key): Promise<boolean> => {
    keys.push(key);
    return true;
  });
  assert.deepEqual(keys.sort(), ["a", "b"]);
});
