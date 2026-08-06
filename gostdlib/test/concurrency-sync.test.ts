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

  Lock(): Promise<void> {
    return Mutex.Lock(this.#mutex);
  }

  Unlock(): void {
    Mutex.Unlock(this.#mutex);
  }
}

test("Mutex serializes waiters and rejects an unmatched unlock", async () => {
  const mutex = new Mutex();
  const events: string[] = [];

  await Mutex.Lock(mutex);
  const waiter = (async (): Promise<void> => {
    await Mutex.Lock(mutex);
    events.push("waiter");
    Mutex.Unlock(mutex);
  })();
  events.push("owner");
  Mutex.Unlock(mutex);
  await waiter;

  assert.deepEqual(events, ["owner", "waiter"]);
  assert.throws(() => Mutex.Unlock(mutex), panicWith(/unlocked mutex/u));
});

test("Cond Wait releases and reacquires its Locker before returning", async () => {
  const locker = new TestLocker();
  const condition = NewCond(locker);

  await locker.Lock();
  const waiting = Cond.Wait(condition);
  await locker.Lock();
  Cond.Signal(condition);
  locker.Unlock();
  await waiting;
  locker.Unlock();
});

test("Cond Broadcast wakes every registered waiter", async () => {
  const locker = new TestLocker();
  const condition = NewCond(locker);
  let resumed = 0;
  const waitOnce = async (): Promise<void> => {
    await Cond.Wait(condition);
    resumed += 1;
    locker.Unlock();
  };

  await locker.Lock();
  const first = waitOnce();
  await locker.Lock();
  const second = waitOnce();
  await locker.Lock();
  Cond.Broadcast(condition);
  locker.Unlock();
  await Promise.all([first, second]);
  assert.equal(resumed, 2);
});

test("RWMutex allows readers together and gives a queued writer exclusivity", async () => {
  const mutex = new RWMutex();
  await RWMutex.RLock(mutex);
  await RWMutex.RLock(mutex);

  let acquired = false;
  const writer = (async (): Promise<void> => {
    await RWMutex.Lock(mutex);
    acquired = true;
    RWMutex.Unlock(mutex);
  })();

  await Promise.resolve();
  assert.equal(acquired, false);
  RWMutex.RUnlock(mutex);
  RWMutex.RUnlock(mutex);
  await writer;
  assert.equal(acquired, true);
});

test("Once, OnceFunc, and OnceValue evaluate exactly once", async () => {
  const once = new Once();
  let count = 0;
  await Promise.all([
    Once.Do(once, async () => {
      count += 1;
      await Promise.resolve();
    }),
    Once.Do(once, async () => {
      count += 1;
    }),
  ]);

  const onceFunction = OnceFunc(async () => {
    count += 10;
  });
  await Promise.all([onceFunction(), onceFunction()]);

  const onceValue = OnceValue(async () => {
    count += 100;
    await Promise.resolve();
    return count;
  });
  assert.deepEqual(await Promise.all([onceValue(), onceValue()]), [111, 111]);
  assert.equal(count, 111);
});

test("OnceValue replays the first panic", async () => {
  let count = 0;
  const onceValue = OnceValue<number>(() => {
    count += 1;
    GoPanic.raise(new ProviderError("once failure"));
  });

  const first = await onceValue().catch((failure: object): object => failure);
  const second = await onceValue().catch((failure: object): object => failure);
  assert.equal(first, second);
  assert.equal(count, 1);
});

test("WaitGroup waits for Add, Done, and Go work", async () => {
  const group = new WaitGroup();
  WaitGroup.Add(group, 1n);
  const manual = Promise.resolve().then(() => WaitGroup.Done(group));
  WaitGroup.Go(group, async () => {
    await Promise.resolve();
  });

  await WaitGroup.Wait(group);
  await manual;
  assert.throws(
    () => WaitGroup.Done(group),
    panicWith(/negative WaitGroup counter/u),
  );
});

test("sync Map and Pool preserve stored interface values", async () => {
  const key = new ProviderError("key");
  const first = new ProviderError("first");
  const second = new ProviderError("second");
  const values = new SyncMap();

  assert.deepEqual(SyncMap.Load(values, key), [undefined, false]);
  SyncMap.Store(values, key, first);
  assert.deepEqual(SyncMap.LoadOrStore(values, key, second), [first, true]);

  const visited: Array<[object | undefined, object | undefined]> = [];
  await SyncMap.Range(values, async (entryKey, entryValue) => {
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
