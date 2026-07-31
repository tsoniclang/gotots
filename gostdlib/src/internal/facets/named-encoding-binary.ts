import type {
  AppendByteOrder,
  ByteOrder,
} from "../../encoding/binary.js";
import type { gostring } from "@gotots/runtime/scalars.js";

export interface BinaryEndianRepresentation
  extends ByteOrder, AppendByteOrder {
  GoString(): gostring;
}

export type BinaryBigEndianStorage = BinaryEndianRepresentation;

export class BinaryBigEndianOperations {
  static $copy(
    source: BinaryEndianRepresentation,
  ): BinaryEndianRepresentation {
    return source;
  }

  static $equal(
    _left: BinaryEndianRepresentation,
    _right: BinaryEndianRepresentation,
  ): boolean {
    return true;
  }

  static $hash(_source: BinaryEndianRepresentation): number {
    return 0;
  }

  static $storageOf(
    source: BinaryEndianRepresentation,
  ): BinaryBigEndianStorage {
    return source;
  }

  static $fromStorage(
    source: BinaryBigEndianStorage,
  ): BinaryEndianRepresentation {
    return source;
  }
}

export type BinaryLittleEndianStorage = BinaryEndianRepresentation;

export class BinaryLittleEndianOperations {
  static $copy(
    source: BinaryEndianRepresentation,
  ): BinaryEndianRepresentation {
    return source;
  }

  static $equal(
    _left: BinaryEndianRepresentation,
    _right: BinaryEndianRepresentation,
  ): boolean {
    return true;
  }

  static $hash(_source: BinaryEndianRepresentation): number {
    return 0;
  }

  static $storageOf(
    source: BinaryEndianRepresentation,
  ): BinaryLittleEndianStorage {
    return source;
  }

  static $fromStorage(
    source: BinaryLittleEndianStorage,
  ): BinaryEndianRepresentation {
    return source;
  }
}

export type BinaryNativeEndianStorage = BinaryEndianRepresentation;

export class BinaryNativeEndianOperations {
  static $copy(
    source: BinaryEndianRepresentation,
  ): BinaryEndianRepresentation {
    return source;
  }

  static $equal(
    _left: BinaryEndianRepresentation,
    _right: BinaryEndianRepresentation,
  ): boolean {
    return true;
  }

  static $hash(_source: BinaryEndianRepresentation): number {
    return 0;
  }

  static $storageOf(
    source: BinaryEndianRepresentation,
  ): BinaryNativeEndianStorage {
    return source;
  }

  static $fromStorage(
    source: BinaryNativeEndianStorage,
  ): BinaryEndianRepresentation {
    return source;
  }
}
