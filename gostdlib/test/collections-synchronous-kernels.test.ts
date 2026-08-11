import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";

import { GoMap, type GoMapValue } from "@gotots/runtime/map.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  MapsEqualFuncKernel as MapsEqualFunc,
  MapsEqualFuncSynchronousKernel as MapsEqualFuncSynchronous,
} from "../src/internal/facets/generic-maps-kernel.js";
import {
  SlicesBinarySearchFuncKernel as BinarySearchFunc,
  SlicesBinarySearchFuncSynchronousKernel as BinarySearchFuncSynchronous,
  SlicesCompactFuncKernel as CompactFunc,
  SlicesCompactFuncSynchronousKernel as CompactFuncSynchronous,
  SlicesCompareFuncKernel as CompareFunc,
  SlicesCompareFuncSynchronousKernel as CompareFuncSynchronous,
  SlicesContainsFuncKernel as ContainsFunc,
  SlicesContainsFuncSynchronousKernel as ContainsFuncSynchronous,
  SlicesDeleteFuncKernel as DeleteFunc,
  SlicesDeleteFuncSynchronousKernel as DeleteFuncSynchronous,
  SlicesEqualFuncKernel as EqualFunc,
  SlicesEqualFuncSynchronousKernel as EqualFuncSynchronous,
  SlicesIndexFuncKernel as IndexFunc,
  SlicesIndexFuncSynchronousKernel as IndexFuncSynchronous,
} from "../src/internal/facets/generic-slices-kernel.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

interface CollectionResults {
  readonly binary: readonly [string, boolean];
  readonly binaryTrace: readonly string[];
  readonly compare: string;
  readonly compareTrace: readonly string[];
  readonly contains: boolean;
  readonly containsTrace: readonly number[];
  readonly equal: boolean;
  readonly equalTrace: readonly string[];
  readonly index: string;
  readonly indexTrace: readonly number[];
  readonly compact: readonly string[];
  readonly compactSource: readonly string[];
  readonly compactTrace: readonly string[];
  readonly deleted: readonly number[];
  readonly deleteSource: readonly number[];
  readonly deleteTrace: readonly number[];
  readonly mapsEqual: boolean;
  readonly mapsEqualCalls: number;
}

const copyValue = <T>(value: T): T => value;
const sliceValue = <T>(source: RuntimeSlice<T>): RuntimeSlice<T> => source;
const mapValue = <K, V>(source: GoMapValue<K, V>): GoMapValue<K, V> => source;
const zeroNumber = (): number => 0;
const zeroString = (): string => "";

test("synchronous callback kernels agree with Go and canonical kernels", async (): Promise<void> => {
  const expected = goResults();
  assert.deepEqual(await canonicalResults(), expected);
  assert.deepEqual(synchronousResults(), expected);
});

test("synchronous callback kernels preserve canonical nil-call behavior", async (): Promise<void> => {
  const canonicalFailures = await Promise.all([
    captureAsyncFailure(() => ContainsFunc(
      sliceValue,
      copyValue,
      copyValue,
      RuntimeSlice.literal([1]),
      undefined,
    )),
    captureAsyncFailure(() => CompactFunc(
      sliceValue,
      sliceValue,
      copyValue,
      copyValue,
      copyValue,
      zeroNumber,
      RuntimeSlice.literal([1, 2]),
      undefined,
    )),
    captureAsyncFailure(() => MapsEqualFunc(
      mapValue,
      mapValue,
      copyValue,
      copyValue,
      GoMap.make<string, number>(0, 0, [["a", 1]]),
      GoMap.make<string, number>(0, 0, [["a", 1]]),
      undefined,
    )),
  ]);
  const synchronousFailures = [
    captureFailure(() => ContainsFuncSynchronous(
      sliceValue,
      copyValue,
      copyValue,
      RuntimeSlice.literal([1]),
      undefined,
    )),
    captureFailure(() => CompactFuncSynchronous(
      sliceValue,
      sliceValue,
      copyValue,
      copyValue,
      copyValue,
      zeroNumber,
      RuntimeSlice.literal([1, 2]),
      undefined,
    )),
    captureFailure(() => MapsEqualFuncSynchronous(
      mapValue,
      mapValue,
      copyValue,
      copyValue,
      GoMap.make<string, number>(0, 0, [["a", 1]]),
      GoMap.make<string, number>(0, 0, [["a", 1]]),
      undefined,
    )),
  ];
  assert.deepEqual(synchronousFailures, canonicalFailures);
  assert.equal(synchronousFailures.every((failure): boolean => failure !== ""), true);
});

async function canonicalResults(): Promise<CollectionResults> {
  const binaryTrace: string[] = [];
  const binary = await BinarySearchFunc<RuntimeSlice<number>, number, number, number>(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 3, 8]),
    3,
    (left, right): bigint => {
      binaryTrace.push(`${left}:${right}`);
      return BigInt(left - right);
    },
  );
  const compareTrace: string[] = [];
  const compare = await CompareFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 5]),
    RuntimeSlice.literal([1, 4, 0]),
    (left: number, right: number): bigint => {
      compareTrace.push(`${left}:${right}`);
      return BigInt(left - right);
    },
  );
  const containsTrace: number[] = [];
  const contains = await ContainsFunc(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 8]),
    (value: number): boolean => {
      containsTrace.push(value);
      return value > 5;
    },
  );
  const equalTrace: string[] = [];
  const equal = await EqualFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 8]),
    RuntimeSlice.literal(["1", "x", "8"]),
    (left: number, right: string): boolean => {
      equalTrace.push(`${left}:${right}`);
      return String(left) === right;
    },
  );
  const indexTrace: number[] = [];
  const index = await IndexFunc(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 8]),
    (value: number): boolean => {
      indexTrace.push(value);
      return value % 2 === 0;
    },
  );
  const compactTrace: string[] = [];
  const compactSource = RuntimeSlice.literal(["A", "a", "B", "b"]);
  const compact = await CompactFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroString,
    compactSource,
    (left, right): boolean => {
      compactTrace.push(`${left}:${right}`);
      return left.toLowerCase() === right.toLowerCase();
    },
  );
  const deleteTrace: number[] = [];
  const deleteSource = RuntimeSlice.literal([1, 2, 3, 4]);
  const deleted = await DeleteFunc(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroNumber,
    deleteSource,
    (value): boolean => {
      deleteTrace.push(value);
      return value % 2 === 0;
    },
  );
  let mapsEqualCalls = 0;
  const mapsEqual = await MapsEqualFunc(
    mapValue,
    mapValue,
    copyValue,
    copyValue,
    GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]),
    GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]),
    (left, right): boolean => {
      mapsEqualCalls += 1;
      return left === right;
    },
  );
  return result(
    binary,
    binaryTrace,
    compare,
    compareTrace,
    contains,
    containsTrace,
    equal,
    equalTrace,
    index,
    indexTrace,
    compact,
    compactSource,
    compactTrace,
    deleted,
    deleteSource,
    deleteTrace,
    mapsEqual,
    mapsEqualCalls,
  );
}

function synchronousResults(): CollectionResults {
  const binaryTrace: string[] = [];
  const binary = BinarySearchFuncSynchronous<RuntimeSlice<number>, number, number, number>(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 3, 8]),
    3,
    (left, right): bigint => {
      binaryTrace.push(`${left}:${right}`);
      return BigInt(left - right);
    },
  );
  const compareTrace: string[] = [];
  const compare = CompareFuncSynchronous(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 5]),
    RuntimeSlice.literal([1, 4, 0]),
    (left: number, right: number): bigint => {
      compareTrace.push(`${left}:${right}`);
      return BigInt(left - right);
    },
  );
  const containsTrace: number[] = [];
  const contains = ContainsFuncSynchronous(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 8]),
    (value: number): boolean => {
      containsTrace.push(value);
      return value > 5;
    },
  );
  const equalTrace: string[] = [];
  const equal = EqualFuncSynchronous(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 8]),
    RuntimeSlice.literal(["1", "x", "8"]),
    (left: number, right: string): boolean => {
      equalTrace.push(`${left}:${right}`);
      return String(left) === right;
    },
  );
  const indexTrace: number[] = [];
  const index = IndexFuncSynchronous(
    sliceValue,
    copyValue,
    copyValue,
    RuntimeSlice.literal([1, 3, 8]),
    (value: number): boolean => {
      indexTrace.push(value);
      return value % 2 === 0;
    },
  );
  const compactTrace: string[] = [];
  const compactSource = RuntimeSlice.literal(["A", "a", "B", "b"]);
  const compact = CompactFuncSynchronous(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroString,
    compactSource,
    (left, right): boolean => {
      compactTrace.push(`${left}:${right}`);
      return left.toLowerCase() === right.toLowerCase();
    },
  );
  const deleteTrace: number[] = [];
  const deleteSource = RuntimeSlice.literal([1, 2, 3, 4]);
  const deleted = DeleteFuncSynchronous(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroNumber,
    deleteSource,
    (value): boolean => {
      deleteTrace.push(value);
      return value % 2 === 0;
    },
  );
  let mapsEqualCalls = 0;
  const mapsEqual = MapsEqualFuncSynchronous(
    mapValue,
    mapValue,
    copyValue,
    copyValue,
    GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]),
    GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]),
    (left, right): boolean => {
      mapsEqualCalls += 1;
      return left === right;
    },
  );
  return result(
    binary,
    binaryTrace,
    compare,
    compareTrace,
    contains,
    containsTrace,
    equal,
    equalTrace,
    index,
    indexTrace,
    compact,
    compactSource,
    compactTrace,
    deleted,
    deleteSource,
    deleteTrace,
    mapsEqual,
    mapsEqualCalls,
  );
}

function result(
  binary: readonly [bigint, boolean],
  binaryTrace: readonly string[],
  compare: bigint,
  compareTrace: readonly string[],
  contains: boolean,
  containsTrace: readonly number[],
  equal: boolean,
  equalTrace: readonly string[],
  index: bigint,
  indexTrace: readonly number[],
  compact: RuntimeSlice<string>,
  compactSource: RuntimeSlice<string>,
  compactTrace: readonly string[],
  deleted: RuntimeSlice<number>,
  deleteSource: RuntimeSlice<number>,
  deleteTrace: readonly number[],
  mapsEqual: boolean,
  mapsEqualCalls: number,
): CollectionResults {
  return {
    binary: [binary[0].toString(), binary[1]],
    binaryTrace,
    compare: compare.toString(),
    compareTrace,
    contains,
    containsTrace,
    equal,
    equalTrace,
    index: index.toString(),
    indexTrace,
    compact: sliceValues(compact),
    compactSource: sliceValues(compactSource),
    compactTrace,
    deleted: sliceValues(deleted),
    deleteSource: sliceValues(deleteSource),
    deleteTrace,
    mapsEqual,
    mapsEqualCalls,
  };
}

function goResults(): CollectionResults {
  const directory = mkdtempSync(join(tmpdir(), "gotots-sync-collections-"));
  const source = join(directory, "main.go");
  try {
    writeFileSync(source, goProgram);
    const completed = spawnSync("go", ["run", source], { encoding: "utf8" });
    assert.equal(completed.status, 0, completed.stderr);
    return JSON.parse(completed.stdout) as CollectionResults;
  } finally {
    rmSync(directory, { force: true, recursive: true });
  }
}

async function captureAsyncFailure(operation: () => Promise<unknown>): Promise<string> {
  try {
    await operation();
    return "";
  } catch (failure: unknown) {
    return String(failure);
  }
}

function captureFailure(operation: () => unknown): string {
  try {
    operation();
    return "";
  } catch (failure: unknown) {
    return String(failure);
  }
}

const goProgram = `
package main

import (
  "encoding/json"
  "fmt"
  "maps"
  "slices"
)

type results struct {
  Binary [2]any \`json:"binary"\`
  BinaryTrace []string \`json:"binaryTrace"\`
  Compare string \`json:"compare"\`
  CompareTrace []string \`json:"compareTrace"\`
  Contains bool \`json:"contains"\`
  ContainsTrace []int \`json:"containsTrace"\`
  Equal bool \`json:"equal"\`
  EqualTrace []string \`json:"equalTrace"\`
  Index string \`json:"index"\`
  IndexTrace []int \`json:"indexTrace"\`
  Compact []string \`json:"compact"\`
  CompactSource []string \`json:"compactSource"\`
  CompactTrace []string \`json:"compactTrace"\`
  Deleted []int \`json:"deleted"\`
  DeleteSource []int \`json:"deleteSource"\`
  DeleteTrace []int \`json:"deleteTrace"\`
  MapsEqual bool \`json:"mapsEqual"\`
  MapsEqualCalls int \`json:"mapsEqualCalls"\`
}

func main() {
  value := results{}
  binaryIndex, binaryFound := slices.BinarySearchFunc([]int{1, 3, 3, 8}, 3, func(left, right int) int {
    value.BinaryTrace = append(value.BinaryTrace, fmt.Sprintf("%d:%d", left, right))
    return left - right
  })
  value.Binary = [2]any{fmt.Sprint(binaryIndex), binaryFound}
  compare := slices.CompareFunc([]int{1, 3, 5}, []int{1, 4, 0}, func(left, right int) int {
    value.CompareTrace = append(value.CompareTrace, fmt.Sprintf("%d:%d", left, right))
    return left - right
  })
  value.Compare = fmt.Sprint(compare)
  value.Contains = slices.ContainsFunc([]int{1, 3, 8}, func(entry int) bool {
    value.ContainsTrace = append(value.ContainsTrace, entry)
    return entry > 5
  })
  value.Equal = slices.EqualFunc([]int{1, 3, 8}, []string{"1", "x", "8"}, func(left int, right string) bool {
    value.EqualTrace = append(value.EqualTrace, fmt.Sprintf("%d:%s", left, right))
    return fmt.Sprint(left) == right
  })
  index := slices.IndexFunc([]int{1, 3, 8}, func(entry int) bool {
    value.IndexTrace = append(value.IndexTrace, entry)
    return entry % 2 == 0
  })
  value.Index = fmt.Sprint(index)
  compactSource := []string{"A", "a", "B", "b"}
  value.Compact = slices.CompactFunc(compactSource, func(left, right string) bool {
    value.CompactTrace = append(value.CompactTrace, left + ":" + right)
    return (left == "A" || left == "a") == (right == "A" || right == "a")
  })
  value.CompactSource = compactSource
  source := []int{1, 2, 3, 4}
  value.Deleted = slices.DeleteFunc(source, func(entry int) bool {
    value.DeleteTrace = append(value.DeleteTrace, entry)
    return entry % 2 == 0
  })
  value.DeleteSource = source
  value.MapsEqual = maps.EqualFunc(
    map[string]int{"a": 1, "b": 2},
    map[string]int{"a": 1, "b": 2},
    func(left, right int) bool {
      value.MapsEqualCalls++
      return left == right
    },
  )
  encoded, failure := json.Marshal(value)
  if failure != nil {
    panic(failure)
  }
  fmt.Println(string(encoded))
}
`;
