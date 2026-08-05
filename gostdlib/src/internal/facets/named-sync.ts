import { Cond, Map, Mutex, Once, Pool, RWMutex, WaitGroup } from "../../sync.js";

export type SyncCondStorage = Cond;

export class SyncCondOperations {
  static $zero(): Cond {
    return new Cond();
  }

  static $copy(source: Cond): Cond {
    return Cond.$copy(source);
  }

  static $equal(left: Cond, right: Cond): boolean {
    return Cond.$equal(left, right);
  }

  static $hash(source: Cond): number {
    return Cond.$hash(source);
  }

  static $storageOf(source: Cond): SyncCondStorage {
    return source;
  }

  static $fromStorage(source: SyncCondStorage): Cond {
    return source;
  }
}

export type SyncMapStorage = Map;

export class SyncMapOperations {
  static $zero(): Map {
    return new Map();
  }

  static $copy(_source: Map): Map {
    return new Map();
  }

  static $storageOf(source: Map): SyncMapStorage {
    return source;
  }

  static $fromStorage(source: SyncMapStorage): Map {
    return source;
  }
}

export type SyncMutexStorage = Mutex;

export class SyncMutexOperations {
  static $zero(): Mutex {
    return new Mutex();
  }

  static $copy(source: Mutex): Mutex {
    return Mutex.$copy(source);
  }

  static $equal(left: Mutex, right: Mutex): boolean {
    return Mutex.$equal(left, right);
  }

  static $hash(source: Mutex): number {
    return Mutex.$hash(source);
  }

  static $storageOf(source: Mutex): SyncMutexStorage {
    return source;
  }

  static $fromStorage(source: SyncMutexStorage): Mutex {
    return source;
  }
}

export type SyncOnceStorage = Once;

export class SyncOnceOperations {
  static $zero(): Once {
    return new Once();
  }

  static $copy(source: Once): Once {
    return Once.$copy(source);
  }

  static $equal(left: Once, right: Once): boolean {
    return Once.$equal(left, right);
  }

  static $hash(source: Once): number {
    return Once.$hash(source);
  }

  static $storageOf(source: Once): SyncOnceStorage {
    return source;
  }

  static $fromStorage(source: SyncOnceStorage): Once {
    return source;
  }
}

export type SyncPoolStorage = Pool;

export class SyncPoolOperations {
  static $zero(): Pool {
    return new Pool();
  }

  static $copy(source: Pool): Pool {
    return new Pool(source.New);
  }

  static $storageOf(source: Pool): SyncPoolStorage {
    return source;
  }

  static $fromStorage(source: SyncPoolStorage): Pool {
    return source;
  }
}

export type SyncRWMutexStorage = RWMutex;

export class SyncRWMutexOperations {
  static $zero(): RWMutex {
    return new RWMutex();
  }

  static $copy(source: RWMutex): RWMutex {
    return RWMutex.$copy(source);
  }

  static $equal(left: RWMutex, right: RWMutex): boolean {
    return RWMutex.$equal(left, right);
  }

  static $hash(source: RWMutex): number {
    return RWMutex.$hash(source);
  }

  static $storageOf(source: RWMutex): SyncRWMutexStorage {
    return source;
  }

  static $fromStorage(source: SyncRWMutexStorage): RWMutex {
    return source;
  }
}

export type SyncWaitGroupStorage = WaitGroup;

export class SyncWaitGroupOperations {
  static $zero(): WaitGroup {
    return new WaitGroup();
  }

  static $copy(source: WaitGroup): WaitGroup {
    return WaitGroup.$copy(source);
  }

  static $equal(left: WaitGroup, right: WaitGroup): boolean {
    return WaitGroup.$equal(left, right);
  }

  static $hash(source: WaitGroup): number {
    return WaitGroup.$hash(source);
  }

  static $storageOf(source: WaitGroup): SyncWaitGroupStorage {
    return source;
  }

  static $fromStorage(source: SyncWaitGroupStorage): WaitGroup {
    return source;
  }
}
