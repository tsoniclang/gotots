import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  SlicesCloneKernel as Clone,
  SlicesConcatKernel as Concat,
  SlicesDeleteKernel as Delete,
  SlicesInsertKernel as Insert,
  SlicesRepeatKernel as Repeat,
  SlicesReplaceKernel as Replace,
  SlicesReverseKernel as Reverse,
} from "../src/internal/facets/generic-slices-kernel.js";

type RecordStorage = { value: number };

class ValueRecord {
  constructor(readonly storage: RecordStorage) {}

  get value(): number {
    return this.storage.value;
  }

  set value(value: number) {
    this.storage.value = value;
  }
}

const sliceValue = (
  source: RuntimeSlice<RecordStorage>,
): RuntimeSlice<RecordStorage> => source;
const copyRecord = (source: ValueRecord): ValueRecord => (
  new ValueRecord({ value: source.value })
);
const fromStorage = (source: RecordStorage): ValueRecord => (
  new ValueRecord(source)
);
const toStorage = (source: ValueRecord): RecordStorage => source.storage;
const zeroRecord = (): ValueRecord => new ValueRecord({ value: 0 });

function records(...values: number[]): RuntimeSlice<RecordStorage> {
  return RuntimeSlice.literal(values.map((value): RecordStorage => ({ value })));
}

function recordsWithCapacity(
  capacity: number,
  ...values: number[]
): RuntimeSlice<RecordStorage> {
  const result = RuntimeSlice.make<RecordStorage>(
    values.length,
    capacity,
    { value: 0 },
  );
  for (const [index, value] of values.entries()) {
    result.set(index, { value });
  }
  return result;
}

function recordValues(source: RuntimeSlice<RecordStorage>): number[] {
  const result: number[] = [];
  for (let index = 0; index < source.length; index += 1) {
    result.push(fromStorage(source.get(index)).value);
  }
  return result;
}

function setRecord(
  source: RuntimeSlice<RecordStorage>,
  index: number,
  value: number,
): void {
  fromStorage(source.get(index)).value = value;
}

test("Clone and Concat assign independent aggregate values", (): void => {
  const source = records(1, 2);
  const clone = Clone(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    source,
  );
  setRecord(clone, 0, 10);
  assert.deepEqual(recordValues(source), [1, 2]);
  assert.deepEqual(recordValues(clone), [10, 2]);

  const right = records(3);
  const concatenated = Concat(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    RuntimeSlice.literal([source, right]),
  );
  setRecord(concatenated, 2, 30);
  assert.deepEqual(recordValues(source), [1, 2]);
  assert.deepEqual(recordValues(right), [3]);
  assert.deepEqual(recordValues(concatenated), [1, 2, 30]);

  assert.equal(Clone(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    RuntimeSlice.nil<RecordStorage>(),
  ).isNil(), true);
  assert.equal(Concat(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    RuntimeSlice.literal([]),
  ).isNil(), true);
});

test("Delete preserves backing aliases and clears the obsolete tail", (): void => {
  const source = records(1, 2, 3, 4);
  const result = Delete(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    source,
    1n,
    3n,
  );
  assert.deepEqual(recordValues(result), [1, 4]);
  assert.deepEqual(recordValues(source), [1, 4, 0, 0]);
  setRecord(result, 1, 40);
  assert.deepEqual(recordValues(source), [1, 40, 0, 0]);
});

test("Insert preserves or replaces backing according to capacity", (): void => {
  const retained = recordsWithCapacity(4, 1, 3);
  const inserted = records(2);
  const retainedResult = Insert(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    retained,
    1n,
    inserted,
  );
  assert.deepEqual(recordValues(retainedResult), [1, 2, 3]);
  assert.deepEqual(recordValues(retained), [1, 2]);
  setRecord(retainedResult, 1, 20);
  assert.deepEqual(recordValues(retained), [1, 20]);
  assert.deepEqual(recordValues(inserted), [2]);

  const allocated = records(1, 3);
  const allocatedResult = Insert(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    allocated,
    1n,
    inserted,
  );
  setRecord(allocatedResult, 0, 10);
  assert.deepEqual(recordValues(allocated), [1, 3]);
  assert.deepEqual(recordValues(allocatedResult), [10, 2, 3]);

  const overlapping = recordsWithCapacity(6, 1, 2, 3);
  const overlapResult = Insert(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    overlapping,
    1n,
    overlapping.slice(0, 2, null),
  );
  assert.deepEqual(recordValues(overlapResult), [1, 1, 2, 2, 3]);
});

test("Repeat copies every aggregate occurrence and is never nil", (): void => {
  const source = records(7);
  const repeated = Repeat(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    source,
    3n,
  );
  setRecord(repeated, 0, 70);
  assert.deepEqual(recordValues(source), [7]);
  assert.deepEqual(recordValues(repeated), [70, 7, 7]);

  const empty = Repeat(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    RuntimeSlice.nil<RecordStorage>(),
    0n,
  );
  assert.equal(empty.isNil(), false);
  assert.equal(empty.length, 0);
  assert.equal(empty.capacity, 0);
});

test("Replace handles overlap, tail clearing, and reallocation", (): void => {
  const shrinking = records(1, 2, 3, 4);
  const shrinkResult = Replace(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    shrinking,
    1n,
    4n,
    shrinking.slice(0, 2, null),
  );
  assert.deepEqual(recordValues(shrinkResult), [1, 1, 2]);
  assert.deepEqual(recordValues(shrinking), [1, 1, 2, 0]);

  const retained = recordsWithCapacity(6, 1, 4);
  const retainedResult = Replace(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    retained,
    1n,
    2n,
    records(2, 3),
  );
  assert.deepEqual(recordValues(retainedResult), [1, 2, 3]);
  assert.deepEqual(recordValues(retained), [1, 2]);

  const allocated = records(1, 4);
  const allocatedResult = Replace(
    sliceValue,
    sliceValue,
    copyRecord,
    fromStorage,
    toStorage,
    zeroRecord,
    allocated,
    1n,
    2n,
    records(2, 3),
  );
  setRecord(allocatedResult, 0, 10);
  assert.deepEqual(recordValues(allocated), [1, 4]);
  assert.deepEqual(recordValues(allocatedResult), [10, 2, 3]);
});

test("Reverse performs value assignments in place", (): void => {
  const source = records(1, 2, 3);
  let copies = 0;
  Reverse(
    sliceValue,
    (value): ValueRecord => {
      copies += 1;
      return copyRecord(value);
    },
    fromStorage,
    toStorage,
    source,
  );
  assert.deepEqual(recordValues(source), [3, 2, 1]);
  assert.ok(copies > 0);
  setRecord(source, 0, 30);
  assert.deepEqual(recordValues(source), [30, 2, 1]);
});
