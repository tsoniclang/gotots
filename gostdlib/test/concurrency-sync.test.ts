import assert from "node:assert/strict";
import test from "node:test";
import { GoPanic } from "@gotots/runtime/panic.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import {
  ProviderError,
  isGoError,
} from "../src/internal/runtime/error.js";
import {
  Cond,
  type Locker,
  Map as SyncMap,
  Mutex,
  NewCond,
  Once,
  OnceFunc,
  OnceValue,
  Pool,
  RWMutex,
  WaitGroup,
} from "../src/sync.js";

const testLockerType = Object.freeze({ comparable: true });

class TestLocker extends ProviderInterfaceValue implements Locker {
  readonly #mutex = new Mutex();

  constructor() {
    super(testLockerType);
  }

  Lock(): void {
    Mutex.Lock(this.#mutex);
  }

  Unlock(): void {
    Mutex.Unlock(this.#mutex);
  }
}

test("Mutex rejects serial blocking and unmatched unlock", () => {
  const mutex = new Mutex();
  Mutex.Lock(mutex);
  assert.throws(
    () => Mutex.Lock(mutex),
    panicWith(/would block under disabled concurrency/u),
  );
  Mutex.Unlock(mutex);
  Mutex.Lock(mutex);
  Mutex.Unlock(mutex);
  assert.throws(() => Mutex.Unlock(mutex), panicWith(/unlocked mutex/u));
});

test("Cond rejects a wait that would block", () => {
  const locker = new TestLocker();
  const condition = NewCond(locker);
  locker.Lock();
  assert.throws(
    () => Cond.Wait(condition),
    panicWith(/would block under disabled concurrency/u),
  );
  Cond.Signal(condition);
  Cond.Broadcast(condition);
  locker.Unlock();
});

test("RWMutex allows readers and rejects a serially blocked writer", () => {
  const mutex = new RWMutex();
  RWMutex.RLock(mutex);
  RWMutex.RLock(mutex);
  assert.throws(
    () => RWMutex.Lock(mutex),
    panicWith(/would block under disabled concurrency/u),
  );
  RWMutex.RUnlock(mutex);
  RWMutex.RUnlock(mutex);
  RWMutex.Lock(mutex);
  RWMutex.Unlock(mutex);
});

test("Once, OnceFunc, and OnceValue evaluate exactly once", () => {
  const once = new Once();
  let count = 0;
  Once.Do(once, () => {
    count += 1;
  });
  Once.Do(once, () => {
    count += 1;
  });

  const onceFunction = OnceFunc(() => {
    count += 10;
  });
  onceFunction();
  onceFunction();

  const onceValue = OnceValue(() => {
    count += 100;
    return count;
  });
  assert.deepEqual([onceValue(), onceValue()], [111, 111]);
  assert.equal(count, 111);
});

test("OnceValue replays the first panic", () => {
  let count = 0;
  const onceValue = OnceValue<number>(() => {
    count += 1;
    GoPanic.raise(new ProviderError("once failure"));
  });

  let first: object | undefined;
  let second: object | undefined;
  try {
    onceValue();
  } catch (failure) {
    first = failure as object;
  }
  try {
    onceValue();
  } catch (failure) {
    second = failure as object;
  }
  assert.equal(first, second);
  assert.equal(count, 1);
});

test("WaitGroup completes inline work and rejects a blocking wait", () => {
  const group = new WaitGroup();
  WaitGroup.Add(group, 1n);
  assert.throws(
    () => WaitGroup.Wait(group),
    panicWith(/would block under disabled concurrency/u),
  );
  WaitGroup.Done(group);
  WaitGroup.Go(group, () => undefined);
  WaitGroup.Wait(group);
  assert.throws(
    () => WaitGroup.Done(group),
    panicWith(/negative WaitGroup counter/u),
  );
});

test("WaitGroup propagates an inline task failure", () => {
  const group = new WaitGroup();
  const failure = new Error("worker failure");
  assert.throws(
    () => WaitGroup.Go(group, () => {
      throw failure;
    }),
    (caught: unknown): boolean => caught === failure,
  );
  WaitGroup.Wait(group);
});

test("sync Map and Pool preserve stored interface values", () => {
  const key = new ProviderError("key");
  const first = new ProviderError("first");
  const second = new ProviderError("second");
  const values = new SyncMap();

  assert.deepEqual(SyncMap.Load(values, key), [undefined, false]);
  SyncMap.Store(values, key, first);
  assert.deepEqual(SyncMap.LoadOrStore(values, key, second), [first, true]);

  const visited: Array<[object | undefined, object | undefined]> = [];
  SyncMap.Range(values, (entryKey, entryValue) => {
    visited.push([entryKey, entryValue]);
    return true;
  });
  assert.deepEqual(visited, [[key, first]]);
  SyncMap.Delete(values, key);
  assert.deepEqual(SyncMap.Load(values, key), [undefined, false]);
  SyncMap.Store(values, key, second);
  SyncMap.Clear(values);
  assert.deepEqual(SyncMap.Load(values, key), [undefined, false]);

  const pool = new Pool(() => second);
  Pool.Put(pool, first);
  assert.equal(Pool.Get(pool), first);
  assert.equal(Pool.Get(pool), second);
});

function panicWith(pattern: RegExp): (failure: object) => boolean {
  return (failure: object): boolean => failure instanceof GoPanic
    && isGoError(failure.value)
    && pattern.test(failure.value.Error());
}
