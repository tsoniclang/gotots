import { Map, Mutex, Once, Pool, RWMutex, WaitGroup } from "../../sync.js";

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

  static $copy(_source: Mutex): Mutex {
    return new Mutex();
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

  static $copy(_source: Once): Once {
    return new Once();
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

  static $copy(_source: RWMutex): RWMutex {
    return new RWMutex();
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

  static $copy(_source: WaitGroup): WaitGroup {
    return new WaitGroup();
  }

  static $storageOf(source: WaitGroup): SyncWaitGroupStorage {
    return source;
  }

  static $fromStorage(source: SyncWaitGroupStorage): WaitGroup {
    return source;
  }
}
