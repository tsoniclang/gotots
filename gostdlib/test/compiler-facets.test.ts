import assert from "node:assert/strict";
import test from "node:test";

import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

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
  ReflectStructFieldOperations,
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
  SyncCondOperations,
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
  BytesBufferWrite,
} from "../src/internal/facets/recovery-io.js";
import {
  SyscallErrnoIs,
  TimeParseErrorError,
  TimeTimeAppendText,
  TimeTimeIsZero,
  TimeTimeMarshalJSON,
  TimeTimeMarshalText,
} from "../src/internal/facets/recovery-value.js";
import { NewReader } from "../src/bufio.js";
import { NewBuffer } from "../src/bytes.js";
import { state as fsState } from "../src/io/fs.js";
import { Description, Sample, Value } from "../src/runtime/metrics.js";
import {
  StructField,
  StructTag,
  Value as ReflectValue,
} from "../src/reflect.js";
import { ParseError, Time } from "../src/time.js";
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
import { Cond, Mutex, Once, Pool, RWMutex, WaitGroup } from "../src/sync.js";
import { EPERM } from "../src/syscall.js";
import { Builder as StringBuilder } from "../src/strings.js";
import { state as binaryState } from "../src/encoding/binary.js";


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
  assert.equal(memStats.Alloc, 0n);
  assert.equal(memStats.EnableGC, false);

  const invalid = ReflectValueOperations.$zero();
  assert.ok(invalid instanceof ReflectValue);
  assert.equal(ReflectValueOperations.$copy(invalid), invalid);

  const field = new StructField({
    Name: "Original",
    PkgPath: "",
    Type: undefined,
    Tag: new StructTag('json:"original"'),
    Offset: 8n,
    Index: RuntimeSlice.literal([1n, 2n]),
    Anonymous: false,
  });
  const fieldCopy = ReflectStructFieldOperations.$copy(field);
  field.Name = "Changed";
  assert.equal(fieldCopy.Name, "Original");
  assert.equal(fieldCopy.Index, field.Index);

  const atomicBool = SyncAtomicBoolOperations.$zero();
  const atomicBoolCopy = SyncAtomicBoolOperations.$copy(atomicBool);
  assert.notEqual(atomicBoolCopy, atomicBool);
  assert.equal(SyncAtomicBoolOperations.$equal(atomicBool, atomicBoolCopy), true);
  assert.equal(
    SyncAtomicBoolOperations.$hash(atomicBool),
    SyncAtomicBoolOperations.$hash(atomicBoolCopy),
  );
  AtomicBool.Store(atomicBool, true);
  assert.equal(SyncAtomicBoolOperations.$equal(atomicBool, atomicBoolCopy), false);
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
  assert.equal(SyncAtomicInt32Operations.$equal(atomicInt32, atomicInt32Copy), true);
  assert.equal(SyncAtomicInt64Operations.$equal(atomicInt64, atomicInt64Copy), true);
  assert.equal(SyncAtomicUint32Operations.$equal(atomicUint32, atomicUint32Copy), true);
  assert.equal(SyncAtomicUint64Operations.$equal(atomicUint64, atomicUint64Copy), true);
  assert.equal(SyncAtomicInt32Operations.$hash(atomicInt32), SyncAtomicInt32Operations.$hash(atomicInt32Copy));
  assert.equal(SyncAtomicInt64Operations.$hash(atomicInt64), SyncAtomicInt64Operations.$hash(atomicInt64Copy));
  assert.equal(SyncAtomicUint32Operations.$hash(atomicUint32), SyncAtomicUint32Operations.$hash(atomicUint32Copy));
  assert.equal(SyncAtomicUint64Operations.$hash(atomicUint64), SyncAtomicUint64Operations.$hash(atomicUint64Copy));
  AtomicInt32.Store(atomicInt32, 1);
  AtomicInt64.Store(atomicInt64, 1n);
  AtomicUint32.Store(atomicUint32, 1);
  AtomicUint64.Add(atomicUint64, 1n);
  assert.equal(SyncAtomicInt32Operations.$equal(atomicInt32, atomicInt32Copy), false);
  assert.equal(SyncAtomicInt64Operations.$equal(atomicInt64, atomicInt64Copy), false);
  assert.equal(SyncAtomicUint32Operations.$equal(atomicUint32, atomicUint32Copy), false);
  assert.equal(SyncAtomicUint64Operations.$equal(atomicUint64, atomicUint64Copy), false);
  assert.equal(AtomicInt32.Load(atomicInt32Copy), 0);
  assert.equal(AtomicInt64.Load(atomicInt64Copy), 0n);
  assert.equal(AtomicUint32.Load(atomicUint32Copy), 0);
  assert.equal(AtomicUint64.Load(atomicUint64Copy), 0n);
  assert.equal(SyncAtomicInt32Operations.$fromStorage(
    SyncAtomicInt32Operations.$storageOf(atomicInt32),
  ), atomicInt32);
  assert.equal(SyncAtomicInt64Operations.$fromStorage(
    SyncAtomicInt64Operations.$storageOf(atomicInt64),
  ), atomicInt64);
  const zeroTime = new Time();
  assert.notEqual(TimeOperations.$copy(zeroTime), zeroTime);
  assert.equal(TimeOperations.$equal(TimeOperations.$copy(zeroTime), zeroTime), true);
  const assignedTime = new Time();
  const assignedTimeValue = new Time(10, 20, 30);
  TimeOperations.$assign(assignedTime, assignedTimeValue);
  assert.equal(TimeOperations.$equal(assignedTime, assignedTimeValue), true);
  assert.notEqual(assignedTime, assignedTimeValue);
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

test("sync value facets preserve comparable state", async (): Promise<void> => {
  const mutex = new Mutex();
  const otherMutex = new Mutex();
  assert.equal(SyncMutexOperations.$equal(mutex, otherMutex), true);
  assert.equal(
    SyncMutexOperations.$hash(mutex),
    SyncMutexOperations.$hash(otherMutex),
  );

  const once = new Once();
  const otherOnce = new Once();
  assert.equal(SyncOnceOperations.$equal(once, otherOnce), true);
  assert.equal(SyncOnceOperations.$hash(once), SyncOnceOperations.$hash(otherOnce));

  const readWrite = new RWMutex();
  const otherReadWrite = new RWMutex();
  assert.equal(SyncRWMutexOperations.$equal(readWrite, otherReadWrite), true);
  assert.equal(
    SyncRWMutexOperations.$hash(readWrite),
    SyncRWMutexOperations.$hash(otherReadWrite),
  );

  const waitGroup = new WaitGroup();
  const otherWaitGroup = new WaitGroup();
  assert.equal(SyncWaitGroupOperations.$equal(waitGroup, otherWaitGroup), true);
  assert.equal(
    SyncWaitGroupOperations.$hash(waitGroup),
    SyncWaitGroupOperations.$hash(otherWaitGroup),
  );

  const condition = new Cond();
  const otherCondition = new Cond();
  assert.equal(SyncCondOperations.$equal(condition, otherCondition), true);
  assert.equal(
    SyncCondOperations.$hash(condition),
    SyncCondOperations.$hash(otherCondition),
  );

  await Mutex.Lock(mutex);
  Mutex.Unlock(mutex);
  await Once.Do(once, (): void => undefined);
  await Once.Do(otherOnce, (): void => undefined);
  await RWMutex.RLock(readWrite);
  RWMutex.RUnlock(readWrite);
  WaitGroup.Add(waitGroup, 1n);
  WaitGroup.Done(waitGroup);
  Cond.Signal(condition);
  Cond.Signal(otherCondition);

  assert.equal(SyncMutexOperations.$equal(mutex, otherMutex), true);
  assert.equal(SyncOnceOperations.$equal(once, otherOnce), true);
  assert.equal(SyncRWMutexOperations.$equal(readWrite, otherReadWrite), true);
  assert.equal(SyncWaitGroupOperations.$equal(waitGroup, otherWaitGroup), true);
  assert.equal(SyncCondOperations.$equal(condition, otherCondition), false);

  const conditionCopy = SyncCondOperations.$copy(condition);
  assert.equal(SyncCondOperations.$equal(condition, conditionCopy), true);
  assert.equal(
    SyncCondOperations.$hash(condition),
    SyncCondOperations.$hash(conditionCopy),
  );
  assert.throws(
    () => Cond.Signal(conditionCopy),
    (failure: unknown): boolean => failure instanceof GoPanic &&
      failure.value.$go$format("v", "", undefined) === "sync.Cond is copied",
  );
});

test("recovery facets preserve the direct provider ABI", (): void => {
  const reader = NewReader({
    Read(destination): [bigint, undefined] {
      destination.set(0, 65);
      return [1n, undefined];
    },
		$go$type: Object.freeze({ comparable: true }),
    $go$methods: new Set<object>(),
    $go$implements(): boolean { return true; },
    $go$equal(other): boolean { return this === other; },
    $go$hash(): number { return 0; },
    $go$formatString: false,
    $go$format(): string { return "reader"; },
  });
  const destination = RuntimeSlice.make<number>(1, 1, 0);
  assert.deepEqual(BufioReaderRead(reader, destination), [1n, undefined]);
  assert.equal(destination.get(0), 65);
  const buffer = NewBuffer(RuntimeSlice.nil<number>());
  assert.deepEqual(
    BytesBufferWrite(buffer, RuntimeSlice.literal([66, 67])),
    [2n, undefined],
  );
  assert.equal(SyscallErrnoIs(EPERM, fsState.ErrPermission), true);
  const parseFailure = new ParseError(
    "2006",
    "bad",
    "2006",
    "bad",
    "",
  );
  assert.equal(TimeParseErrorError(parseFailure), parseFailure.Error());
  const [text, textFailure] = TimeTimeAppendText(
    new Time(),
    RuntimeSlice.literal([0x70, 0x3d]),
  );
  assert.equal(textFailure, undefined);
  assert.equal(TimeTimeIsZero(new Time()), true);
  const [json, jsonFailure] = TimeTimeMarshalJSON(new Time());
  assert.equal(jsonFailure, undefined);
  assert.equal(
    new TextDecoder().decode(Uint8Array.from(
      Array.from({ length: json.length }, (_, index) => json.get(index)),
    )),
    '"0001-01-01T00:00:00Z"',
  );
  const [marshaled, marshalFailure] = TimeTimeMarshalText(new Time());
  assert.equal(marshalFailure, undefined);
  assert.equal(
    new TextDecoder().decode(Uint8Array.from(
      Array.from(
        { length: marshaled.length },
        (_, index) => marshaled.get(index),
      ),
    )),
    "0001-01-01T00:00:00Z",
  );
  assert.equal(
    new TextDecoder().decode(Uint8Array.from(
      Array.from({ length: text.length }, (_, index) => text.get(index)),
    )),
    "p=0001-01-01T00:00:00Z",
  );
});
