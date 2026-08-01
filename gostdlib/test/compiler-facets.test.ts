import assert from "node:assert/strict";
import test from "node:test";

import { GoMap, type GoMapValue } from "@gotots/runtime/map.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  MapsKeysCooperative,
} from "../src/internal/facets/generic-maps.js";
import {
  MathBigFloatOperations,
  MathBigIntOperations,
} from "../src/internal/facets/named-math-big.js";
import {
  BinaryBigEndianOperations,
  BinaryLittleEndianOperations,
  BinaryNativeEndianOperations,
} from "../src/internal/facets/named-encoding-binary.js";
import {
  SlicesAppendSeqCooperative,
  SlicesAppendSeqFullyCooperative,
  SlicesCollectCooperative,
  SlicesSortedCooperative,
  SlicesSortedFullyCooperative,
  SlicesValuesCooperative,
  SlicesValuesFullyCooperative,
} from "../src/internal/facets/generic-slices.js";
import {
  RuntimeMetricsDescriptionOperations,
  RuntimeMetricsSampleOperations,
} from "../src/internal/facets/named-runtime-metrics.js";
import {
  StringsBuilderOperations,
} from "../src/internal/facets/named-strings.js";
import {
  RuntimeMemStatsOperations,
} from "../src/internal/facets/named-runtime.js";
import {
  ReflectValueOperations,
} from "../src/internal/facets/named-reflect.js";
import {
  SyncAtomicBoolOperations,
  SyncAtomicInt32Operations,
  SyncAtomicInt64Operations,
  SyncAtomicUint32Operations,
  SyncAtomicUint64Operations,
} from "../src/internal/facets/named-sync-atomic.js";
import {
  SyncMapOperations,
  SyncMutexOperations,
  SyncOnceOperations,
  SyncPoolOperations,
  SyncRWMutexOperations,
  SyncWaitGroupOperations,
} from "../src/internal/facets/named-sync.js";
import { TimeOperations } from "../src/internal/facets/named-time.js";
import {
  UnicodeRange16Operations,
} from "../src/internal/facets/named-unicode.js";
import {
  BufioReaderRead,
} from "../src/internal/facets/recovery-io.js";
import { NewReader } from "../src/bufio.js";
import { Description, Sample, Value } from "../src/runtime/metrics.js";
import { Value as ReflectValue } from "../src/reflect.js";
import { Time } from "../src/time.js";
import { Seq } from "../src/iter.js";
import {
  Accuracy,
  Float as BigFloat,
  Int as BigInteger,
} from "../src/math/big.js";
import {
  Bool as AtomicBool,
  Int32 as AtomicInt32,
  Int64 as AtomicInt64,
  Uint32 as AtomicUint32,
  Uint64 as AtomicUint64,
} from "../src/sync/atomic.js";
import { Pool } from "../src/sync.js";
import { Builder as StringBuilder } from "../src/strings.js";
import { state as binaryState } from "../src/encoding/binary.js";

const copyValue = <T>(value: T): T => value;
const equalValue = <T>(left: T, right: T): boolean => left === right;
const lessNumber = (left: number, right: number): boolean => left < right;
const sliceValue = <T>(value: RuntimeSlice<T>): RuntimeSlice<T> => value;
const zeroNumber = (): number => 0;

test("named-struct facets expose only selected static operations", (): void => {
  assert.equal(BinaryBigEndianOperations.$fromStorage(
    BinaryBigEndianOperations.$storageOf(binaryState.BigEndian),
  ), binaryState.BigEndian);
  assert.equal(
    BinaryBigEndianOperations.$copy(binaryState.BigEndian),
    binaryState.BigEndian,
  );
  assert.equal(BinaryBigEndianOperations.$equal(
    binaryState.BigEndian,
    binaryState.BigEndian,
  ), true);
  assert.equal(BinaryBigEndianOperations.$hash(binaryState.BigEndian), 0);
  assert.equal(BinaryNativeEndianOperations.$fromStorage(
    BinaryNativeEndianOperations.$storageOf(binaryState.NativeEndian),
  ), binaryState.NativeEndian);
  assert.equal(BinaryLittleEndianOperations.$fromStorage(
    BinaryLittleEndianOperations.$storageOf(binaryState.LittleEndian),
  ), binaryState.LittleEndian);
  assert.equal(
    BinaryLittleEndianOperations.$copy(binaryState.LittleEndian),
    binaryState.LittleEndian,
  );
  assert.equal(BinaryNativeEndianOperations.$equal(
    binaryState.NativeEndian,
    binaryState.NativeEndian,
  ), true);
  assert.equal(BinaryLittleEndianOperations.$hash(binaryState.LittleEndian), 0);
  const integer = MathBigIntOperations.$zero();
  assert.equal(BigInteger.String(integer), "0");
  const floating = MathBigFloatOperations.$zero();
  assert.deepEqual(BigFloat.Float64(floating), [0, new Accuracy(0)]);
  const range = UnicodeRange16Operations.$make(1, 4, 1);
  assert.deepEqual([range.Lo, range.Hi, range.Stride], [1, 4, 1]);

  const pool = SyncPoolOperations.$zero();
  assert.equal(SyncPoolOperations.$fromStorage(
    SyncPoolOperations.$storageOf(pool),
  ), pool);

  const mutex = SyncMutexOperations.$zero();
  assert.equal(SyncMutexOperations.$fromStorage(
    SyncMutexOperations.$storageOf(mutex),
  ), mutex);

  const builder = StringsBuilderOperations.$zero();
  const builderCopy = StringsBuilderOperations.$copy(builder);
  assert.notEqual(builderCopy, builder);
  StringBuilder.WriteString(builder, "source");
  assert.equal(StringBuilder.String(builderCopy), "");
  assert.equal(StringsBuilderOperations.$fromStorage(
    StringsBuilderOperations.$storageOf(builder),
  ), builder);

  const sample = RuntimeMetricsSampleOperations.$copy(
    new Sample("/metric", new Value()),
  );
  assert.equal(sample.Name, "/metric");
  const description = RuntimeMetricsDescriptionOperations.$copy(
    new Description("/metric", "detail"),
  );
  assert.equal(description.Description, "detail");

  const memStats = RuntimeMemStatsOperations.$zero();
  assert.equal(memStats.Alloc, 0);
  assert.equal(memStats.EnableGC, false);

  const invalid = ReflectValueOperations.$zero();
  assert.ok(invalid instanceof ReflectValue);
  assert.equal(ReflectValueOperations.$copy(invalid), invalid);

  const atomicBool = SyncAtomicBoolOperations.$zero();
  const atomicBoolCopy = SyncAtomicBoolOperations.$copy(atomicBool);
  assert.notEqual(atomicBoolCopy, atomicBool);
  AtomicBool.Store(atomicBool, true);
  assert.equal(AtomicBool.Load(atomicBoolCopy), false);
  assert.equal(SyncAtomicBoolOperations.$fromStorage(
    SyncAtomicBoolOperations.$storageOf(atomicBool),
  ), atomicBool);
  const atomicInt32 = SyncAtomicInt32Operations.$zero();
  const atomicInt64 = SyncAtomicInt64Operations.$zero();
  const atomicUint32 = SyncAtomicUint32Operations.$zero();
  const atomicUint64 = SyncAtomicUint64Operations.$zero();
  const atomicInt32Copy = SyncAtomicInt32Operations.$copy(atomicInt32);
  const atomicInt64Copy = SyncAtomicInt64Operations.$copy(atomicInt64);
  const atomicUint32Copy = SyncAtomicUint32Operations.$copy(atomicUint32);
  const atomicUint64Copy = SyncAtomicUint64Operations.$copy(atomicUint64);
  AtomicInt32.Store(atomicInt32, 1);
  AtomicInt64.Store(atomicInt64, 1);
  AtomicUint32.Store(atomicUint32, 1);
  assert.equal(AtomicInt32.Load(atomicInt32Copy), 0);
  assert.equal(AtomicInt64.Load(atomicInt64Copy), 0);
  assert.equal(AtomicUint32.Load(atomicUint32Copy), 0);
  assert.equal(AtomicUint64.Load(atomicUint64Copy), 0);
  assert.equal(SyncAtomicInt32Operations.$fromStorage(
    SyncAtomicInt32Operations.$storageOf(atomicInt32),
  ), atomicInt32);
  assert.equal(SyncAtomicInt64Operations.$fromStorage(
    SyncAtomicInt64Operations.$storageOf(atomicInt64),
  ), atomicInt64);
  const zeroTime = new Time();
  assert.equal(TimeOperations.$copy(zeroTime), zeroTime);
  assert.equal(TimeOperations.$fromStorage(
    TimeOperations.$storageOf(zeroTime),
  ), zeroTime);
  assert.equal(TimeOperations.$zero().IsZero(), true);
  assert.equal(TimeOperations.$equal(new Time(), new Time()), true);
  assert.equal(TimeOperations.$equal(
    new Time(10, 20, 30),
    new Time(10, 21, 30),
  ), false);
  assert.equal(
    TimeOperations.$hash(new Time(10, 20, 30)),
    TimeOperations.$hash(new Time(10, 20, 30)),
  );
  assert.notEqual(
    TimeOperations.$hash(new Time(10, 20, 30)),
    TimeOperations.$hash(new Time(10, 21, 30)),
  );
  const waitGroup = SyncWaitGroupOperations.$zero();
  const waitGroupCopy = SyncWaitGroupOperations.$copy(waitGroup);
  assert.notEqual(waitGroupCopy, waitGroup);
  assert.equal(SyncWaitGroupOperations.$fromStorage(
    SyncWaitGroupOperations.$storageOf(waitGroup),
  ), waitGroup);
  const once = SyncOnceOperations.$zero();
  assert.notEqual(SyncOnceOperations.$copy(once), once);
  assert.equal(SyncOnceOperations.$fromStorage(
    SyncOnceOperations.$storageOf(once),
  ), once);
  const copiedMutexSource = SyncMutexOperations.$zero();
  assert.notEqual(
    SyncMutexOperations.$copy(copiedMutexSource),
    copiedMutexSource,
  );
  const readWriteMutex = SyncRWMutexOperations.$zero();
  assert.notEqual(SyncRWMutexOperations.$copy(readWriteMutex), readWriteMutex);
  const concurrentMap = SyncMapOperations.$zero();
  assert.notEqual(SyncMapOperations.$copy(concurrentMap), concurrentMap);
  const poolNew = (): undefined => undefined;
  const copiedPoolSource = new Pool(poolNew);
  const poolCopy = SyncPoolOperations.$copy(copiedPoolSource);
  assert.notEqual(poolCopy, copiedPoolSource);
  assert.equal(poolCopy.New, poolNew);
});

test("recovery facets preserve the direct provider ABI", (): void => {
  const reader = NewReader({
    Read(destination): [number, undefined] {
      destination.set(0, 65);
      return [1, undefined];
    },
    $go$type: Object.freeze({}),
    $go$methods: new Set<object>(),
    $go$implements(): boolean { return true; },
    $go$equal(other): boolean { return this === other; },
    $go$hash(): number { return 0; },
    $go$formatString: false,
    $go$format(): string { return "reader"; },
  });
  const destination = RuntimeSlice.make<number>(1, 1, 0);
  assert.deepEqual(BufioReaderRead(reader, destination), [1, undefined]);
  assert.equal(destination.get(0), 65);
});

test("generic facets adapt cooperative provider implementations", async (): Promise<void> => {
  const source = GoMap.make<string, number>(0, 0, [["a", 1], ["b", 2]]);
  const keys: string[] = [];
  await MapsKeysCooperative<GoMapValue<string, number>, string, number>(
    (value): GoMapValue<string, number> => value,
    (value): string => value,
    source,
  ).value?.(async (key): Promise<boolean> => {
    keys.push(key);
    return true;
  });
  assert.deepEqual(keys.sort(), ["a", "b"]);

  const sequence = new Seq<number, (
    yieldValue: ((value: number) => Promise<boolean>) | undefined,
  ) => Promise<void>>(async (yieldValue): Promise<void> => {
    await yieldValue?.(2);
    await yieldValue?.(3);
  });
  const values = await SlicesCollectCooperative(
    copyValue,
    copyValue,
    sequence,
  );
  assert.deepEqual([values.get(0), values.get(1)], [2, 3]);

  const slice = RuntimeSlice.literal([3, 1, 2]);
  const yielded: number[] = [];
  await SlicesValuesCooperative<RuntimeSlice<number>, number, number>(
    sliceValue,
    copyValue,
    copyValue,
    slice,
  ).value?.(
    async (value): Promise<boolean> => {
      yielded.push(value);
      return true;
    },
  );
  await SlicesValuesFullyCooperative<RuntimeSlice<number>, number, number>(
    sliceValue,
    copyValue,
    copyValue,
    slice,
  ).value?.(
    async (value): Promise<boolean> => {
      yielded.push(value);
      return true;
    },
  );
  assert.deepEqual(yielded, [3, 1, 2, 3, 1, 2]);

  const appended = await SlicesAppendSeqCooperative(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroNumber,
    RuntimeSlice.literal([1]),
    sequence,
  );
  const fullyAppended = await SlicesAppendSeqFullyCooperative(
    sliceValue,
    sliceValue,
    copyValue,
    copyValue,
    copyValue,
    zeroNumber,
    RuntimeSlice.literal([1]),
    sequence,
  );
  assert.deepEqual(
    [appended.get(0), appended.get(1), appended.get(2)],
    [1, 2, 3],
  );
  assert.deepEqual(
    [fullyAppended.get(0), fullyAppended.get(1), fullyAppended.get(2)],
    [1, 2, 3],
  );

  const sorted = await SlicesSortedCooperative<number, number>(
    lessNumber,
    copyValue,
    equalValue,
    copyValue,
    copyValue,
    sequence,
  );
  const fullySorted = await SlicesSortedFullyCooperative<number, number>(
    lessNumber,
    copyValue,
    equalValue,
    copyValue,
    copyValue,
    sequence,
  );
  assert.deepEqual([sorted.get(0), sorted.get(1)], [2, 3]);
  assert.deepEqual([fullySorted.get(0), fullySorted.get(1)], [2, 3]);
});
