import { Map, Mutex, Once, Pool, RWMutex } from "../../sync.js";

export type SyncMapStorage = Map;

export class SyncMapOperations {
  static $zero(): Map {
    return new Map();
  }

  static $storageOf(source: Map): SyncMapStorage {
    return source;
  }

  static $fromStorage(source: SyncMapStorage): Map {
    return source;
  }
}

export class SyncMutexOperations {
  static $zero(): Mutex {
    return new Mutex();
  }
}

export class SyncOnceOperations {
  static $zero(): Once {
    return new Once();
  }
}

export type SyncPoolStorage = Pool;

export class SyncPoolOperations {
  static $zero(): Pool {
    return new Pool();
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

  static $storageOf(source: RWMutex): SyncRWMutexStorage {
    return source;
  }

  static $fromStorage(source: SyncRWMutexStorage): RWMutex {
    return source;
  }
}
