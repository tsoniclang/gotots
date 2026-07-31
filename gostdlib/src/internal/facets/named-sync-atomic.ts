import { Bool, Int32, Int64, Uint32, Uint64 } from "../../sync/atomic.js";

export type SyncAtomicBoolStorage = Bool;

export class SyncAtomicBoolOperations {
  static $zero(): Bool {
    return new Bool();
  }

  static $copy(source: Bool): Bool {
    return new Bool(Bool.Load(source));
  }

  static $storageOf(source: Bool): SyncAtomicBoolStorage {
    return source;
  }

  static $fromStorage(source: SyncAtomicBoolStorage): Bool {
    return source;
  }
}

export type SyncAtomicInt32Storage = Int32;

export class SyncAtomicInt32Operations {
  static $zero(): Int32 {
    return new Int32();
  }

  static $copy(source: Int32): Int32 {
    return new Int32(Int32.Load(source));
  }

  static $storageOf(source: Int32): SyncAtomicInt32Storage {
    return source;
  }

  static $fromStorage(source: SyncAtomicInt32Storage): Int32 {
    return source;
  }
}

export type SyncAtomicInt64Storage = Int64;

export class SyncAtomicInt64Operations {
  static $zero(): Int64 {
    return new Int64();
  }

  static $copy(source: Int64): Int64 {
    return new Int64(Int64.Load(source));
  }

  static $storageOf(source: Int64): SyncAtomicInt64Storage {
    return source;
  }

  static $fromStorage(source: SyncAtomicInt64Storage): Int64 {
    return source;
  }
}

export type SyncAtomicUint32Storage = Uint32;

export class SyncAtomicUint32Operations {
  static $zero(): Uint32 {
    return new Uint32();
  }

  static $copy(source: Uint32): Uint32 {
    return new Uint32(Uint32.Load(source));
  }

  static $storageOf(source: Uint32): SyncAtomicUint32Storage {
    return source;
  }

  static $fromStorage(source: SyncAtomicUint32Storage): Uint32 {
    return source;
  }
}

export type SyncAtomicUint64Storage = Uint64;

export class SyncAtomicUint64Operations {
  static $zero(): Uint64 {
    return new Uint64();
  }

  static $copy(source: Uint64): Uint64 {
    return new Uint64(Uint64.Load(source));
  }

  static $storageOf(source: Uint64): SyncAtomicUint64Storage {
    return source;
  }

  static $fromStorage(source: SyncAtomicUint64Storage): Uint64 {
    return source;
  }
}
