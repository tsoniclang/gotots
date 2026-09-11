import { GoString } from "@gotots/runtime/string-value.js";
import assert from "node:assert/strict";
import test from "node:test";
import { ProviderError, isGoError } from "../src/internal/runtime/error.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { Cond, Map, Mutex, Pool, RWMutex, Once, WaitGroup } from "../src/sync.js";
import { SyncCondOperations, SyncMapOperations, SyncPoolOperations, SyncOnceOperations, SyncRWMutexOperations, SyncWaitGroupOperations } from "../src/internal/facets/named-sync.js";
import { Bool, Int32, Int64, Uint32, Uint64 } from "../src/sync/atomic.js";
import { SyncAtomicBoolOperations, SyncAtomicInt32Operations, SyncAtomicInt64Operations, SyncAtomicUint32Operations, SyncAtomicUint64Operations } from "../src/internal/facets/named-sync-atomic.js";
import { RuntimeMemStatsOperations } from "../src/internal/facets/named-runtime.js";
import { MemStats } from "../src/runtime.js";
import { Description, Read, Sample, Value } from "../src/runtime/metrics.js";
import { RuntimeMetricsDescriptionOperations, RuntimeMetricsSampleOperations } from "../src/internal/facets/named-runtime-metrics.js";
import { StructField, StructTag } from "../src/reflect.js";
import { ReflectStructFieldOperations } from "../src/internal/facets/named-reflect.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { state as binary } from "../src/encoding/binary.js";
import { BinaryBigEndianOperations, BinaryLittleEndianOperations, BinaryNativeEndianOperations } from "../src/internal/facets/named-encoding-binary.js";

test("sync reset preserves the destination and rejects invalid copy sources", () => {
  const source = new Map();
  const target = SyncMapOperations.$copy(source);
  const retained = target;
  const key = new ProviderError(GoString.fromText("key"));
  const value = new ProviderError(GoString.fromText("value"));
  Map.Store(target, key, value);
  SyncMapOperations.$assign(target, source);
  assert.equal(target, retained);
  assert.deepEqual(Map.Load(retained, key), [undefined, false]);
  assert.throws(() => SyncMapOperations.$copy(target), panicWith(/must not be copied after first use/));
  SyncMapOperations.$assign(target, target);
  const poolSource = new Pool(() => value);
  const pool = SyncPoolOperations.$copy(poolSource);
  Pool.Put(pool, key);
  SyncPoolOperations.$assign(pool, poolSource);
  assert.equal(Pool.Get(pool), value);
  assert.throws(() => SyncPoolOperations.$copy(pool), panicWith(/must not be copied after first use/));
});

test("sync state payload assignment retains the receiver location", () => {
  const once = new Once();
  let count = 0;
  Once.Do(once, () => { count++; });
  SyncOnceOperations.$assign(once, new Once());
  Once.Do(once, () => { count++; });
  assert.equal(count, 2);
  const readers = new RWMutex();
  RWMutex.RLock(readers);
  SyncRWMutexOperations.$assign(readers, new RWMutex());
  RWMutex.Lock(readers);
  RWMutex.Unlock(readers);
  const group = new WaitGroup();
  WaitGroup.Add(group, 3n);
  SyncWaitGroupOperations.$assign(group, new WaitGroup());
  WaitGroup.Wait(group);
  const mutex = new Mutex();
  Mutex.Lock(mutex);
  Mutex.$assign(mutex, new Mutex());
  Mutex.Lock(mutex);
  Mutex.Unlock(mutex);
});

test("Cond assignment copies its checker state but not its location identity", () => {
  const original = new Cond();
  Cond.Signal(original);
  const target = new Cond();
  SyncCondOperations.$assign(target, original);
  assert.throws(() => Cond.Signal(target), panicWith(/sync.Cond is copied/));
  SyncCondOperations.$assign(target, new Cond());
  Cond.Signal(target);
  SyncCondOperations.$assign(target, target);
  Cond.Signal(target);
});

test("StructField assignment copies the descriptor and shares its slice data", () => {
  const field = (name: string, index: bigint) => new StructField({
    Name: GoString.fromText(name), PkgPath: GoString.empty, Type: undefined, Tag: new StructTag(GoString.fromText("")),
    Offset: 0n, Index: RuntimeSlice.literal([index]), Anonymous: false,
  });
  const target = field("before", 1n);
  const oldIndex = target.Index;
  const source = field("after", 2n);
  ReflectStructFieldOperations.$assign(target, source);
  assert.equal(target.Name.text(), "after");
  assert.equal(oldIndex.get(0), 1n);
  assert.equal(target.Index, source.Index);
  source.Index.set(0, 3n);
  assert.equal(target.Index.get(0), 3n);
  const copied = ReflectStructFieldOperations.$copy(source);
  source.Name = GoString.fromText("changed");
  assert.equal((copied.Name)?.text(), "after");
});

test("zero-sized endian assignment preserves the selected implementation", () => {
  const selected = [binary.BigEndian, binary.LittleEndian, binary.NativeEndian];
  const before = selected.map(value => value.GoString());
  BinaryBigEndianOperations.$assign(binary.BigEndian, binary.BigEndian);
  BinaryLittleEndianOperations.$assign(binary.LittleEndian, binary.LittleEndian);
  BinaryNativeEndianOperations.$assign(binary.NativeEndian, binary.NativeEndian);
  assert.deepEqual(selected.map(value => value.GoString()), before);
});

test("atomic assignment updates the existing scalar cell", () => {
  const boolean = new Bool(true);
  SyncAtomicBoolOperations.$assign(boolean, new Bool(false));
  assert.equal(Bool.Load(boolean), false);
  const signed32 = new Int32(7);
  SyncAtomicInt32Operations.$assign(signed32, new Int32(-3));
  assert.equal(Int32.Load(signed32), -3);
  const signed64 = new Int64(7n);
  SyncAtomicInt64Operations.$assign(signed64, new Int64(-3n));
  assert.equal(Int64.Load(signed64), -3n);
  const unsigned32 = new Uint32(7);
  SyncAtomicUint32Operations.$assign(unsigned32, new Uint32(3));
  assert.equal(Uint32.Load(unsigned32), 3);
  const unsigned64 = new Uint64(7n);
  SyncAtomicUint64Operations.$assign(unsigned64, new Uint64(3n));
  assert.equal(Uint64.Load(unsigned64), 3n);
});

test("MemStats assignment retains all array and element locations", () => {
  const target = new MemStats();
  const pauses = target.PauseNs;
  const bySize = target.BySize;
  const entry = bySize.get(0);
  const source = new MemStats();
  source.Alloc = 5n;
  source.PauseNs.set(0, 7n);
  source.PauseEnd.set(255, 8n);
  source.BySize.get(0).Size = 9;
  source.BySize.get(0).Mallocs = 10n;
  RuntimeMemStatsOperations.$assign(target, source);
  assert.equal(target.Alloc, 5n);
  assert.equal(target.PauseNs, pauses);
  assert.equal(pauses.get(0), 7n);
  assert.equal(target.PauseEnd.get(255), 8n);
  assert.equal(target.BySize, bySize);
  assert.equal(bySize.get(0), entry);
  assert.equal(entry.Size, 9);
  assert.equal(entry.Mallocs, 10n);
  const copied = RuntimeMemStatsOperations.$copy(source);
  assert.notEqual(copied.BySize.get(0), source.BySize.get(0));
  copied.BySize.get(0).Size = 11;
  assert.equal(source.BySize.get(0).Size, 9);
  RuntimeMemStatsOperations.$assign(target, target);
  assert.equal(entry.Size, 9);
});

test("metrics assignment copies nested values without retargeting retained fields", () => {
  const initial = Value.FromUint64(3n);
  const target = new Sample(GoString.fromText("before"), initial);
  const retained = target.Value;
  const source = new Sample(GoString.fromText("after"), Value.FromUint64(7n));
  RuntimeMetricsSampleOperations.$assign(target, source);
  assert.equal(target.Value, retained);
  assert.equal(retained.Uint64(), 7n);
  assert.equal(initial.Uint64(), 3n);
  const copied = RuntimeMetricsSampleOperations.$copy(source);
  RuntimeMetricsSampleOperations.$assign(source, new Sample(GoString.fromText("again"), Value.FromUint64(9n)));
  assert.equal(copied.Value.Uint64(), 7n);
  const description = new Description(GoString.fromText("before"));
  RuntimeMetricsDescriptionOperations.$assign(description, new Description(GoString.fromText("after")));
  assert.equal(description.Name.text(), "after");
});

test("metrics Read updates a retained Value field instead of replacing it", () => {
  const sample = new Sample(GoString.fromText("/gc/gogc:percent"));
  const retained = sample.Value;
  const samples = RuntimeSlice.literal([sample]);
  Read(samples);
  assert.equal(sample.Value, retained);
  assert.equal(retained.Uint64(), 100n);
  sample.Name = GoString.fromText("not-a-selected-metric");
  Read(samples);
  assert.equal(sample.Value, retained);
  assert.equal(retained.Kind().value, 0n);
});

function panicWith(pattern: RegExp): (failure: object) => boolean {
  return failure => failure instanceof GoPanic && isGoError(failure.value) && pattern.test(failure.value.Error().text());
}
