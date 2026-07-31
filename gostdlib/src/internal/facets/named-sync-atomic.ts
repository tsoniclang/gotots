import { Uint32, Uint64 } from "../../sync/atomic.js";

export type SyncAtomicUint32Storage = Uint32;

export class SyncAtomicUint32Operations {
  static $zero(): Uint32 {
    return new Uint32();
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

  static $storageOf(source: Uint64): SyncAtomicUint64Storage {
    return source;
  }

  static $fromStorage(source: SyncAtomicUint64Storage): Uint64 {
    return source;
  }
}
