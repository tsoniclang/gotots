import type {
  bool,
  int32,
  int64,
  uint32,
  uint64,
} from "@gotots/runtime/scalars.js";
import { GoPanic } from "@gotots/runtime/panic.js";

abstract class Cell<T> {
  protected constructor(protected value: T) {}
}

export class Bool extends Cell<bool> {
  constructor(value: bool = false) {
    super(value);
  }

  static CompareAndSwap(receiver: Bool | undefined, old: bool, next: bool): bool {
    const cell = requireCell(receiver, "Bool.CompareAndSwap");
    if (cell.value !== old) {
      return false;
    }
    cell.value = next;
    return true;
  }

  static Load(receiver: Bool | undefined): bool {
    return requireCell(receiver, "Bool.Load").value;
  }

  static Store(receiver: Bool | undefined, val: bool): void {
    requireCell(receiver, "Bool.Store").value = val;
  }
}

export class Int32 extends Cell<int32> {
  constructor(value: int32 = 0) {
    super(toInt32(value));
  }

  static Add(receiver: Int32 | undefined, delta: int32): int32 {
    const cell = requireCell(receiver, "Int32.Add");
    cell.value = toInt32(cell.value + delta);
    return cell.value;
  }

  static CompareAndSwap(receiver: Int32 | undefined, old: int32, next: int32): bool {
    const cell = requireCell(receiver, "Int32.CompareAndSwap");
    if (cell.value !== toInt32(old)) {
      return false;
    }
    cell.value = toInt32(next);
    return true;
  }

  static Load(receiver: Int32 | undefined): int32 {
    return requireCell(receiver, "Int32.Load").value;
  }

  static Store(receiver: Int32 | undefined, val: int32): void {
    requireCell(receiver, "Int32.Store").value = toInt32(val);
  }

  static Swap(receiver: Int32 | undefined, next: int32): int32 {
    const cell = requireCell(receiver, "Int32.Swap");
    const previous = cell.value;
    cell.value = toInt32(next);
    return previous;
  }
}

export class Int64 extends Cell<int64> {
  constructor(value: int64 = 0) {
    super(value);
  }

  static Add(receiver: Int64 | undefined, delta: int64): int64 {
    const cell = requireCell(receiver, "Int64.Add");
    cell.value += delta;
    return cell.value;
  }

  static CompareAndSwap(receiver: Int64 | undefined, old: int64, next: int64): bool {
    const cell = requireCell(receiver, "Int64.CompareAndSwap");
    if (cell.value !== old) {
      return false;
    }
    cell.value = next;
    return true;
  }

  static Load(receiver: Int64 | undefined): int64 {
    return requireCell(receiver, "Int64.Load").value;
  }

  static Store(receiver: Int64 | undefined, val: int64): void {
    requireCell(receiver, "Int64.Store").value = val;
  }
}

export class Uint32 extends Cell<uint32> {
  constructor(value: uint32 = 0) {
    super(toUint32(value));
  }

  static Add(receiver: Uint32 | undefined, delta: uint32): uint32 {
    const cell = requireCell(receiver, "Uint32.Add");
    cell.value = toUint32(cell.value + delta);
    return cell.value;
  }

  static Load(receiver: Uint32 | undefined): uint32 {
    return requireCell(receiver, "Uint32.Load").value;
  }

  static Store(receiver: Uint32 | undefined, val: uint32): void {
    requireCell(receiver, "Uint32.Store").value = toUint32(val);
  }
}

export class Uint64 extends Cell<uint64> {
  constructor(value: uint64 = 0) {
    super(value);
  }

  static Add(receiver: Uint64 | undefined, delta: uint64): uint64 {
    const cell = requireCell(receiver, "Uint64.Add");
    cell.value += delta;
    return cell.value;
  }

  static CompareAndSwap(receiver: Uint64 | undefined, old: uint64, next: uint64): bool {
    const cell = requireCell(receiver, "Uint64.CompareAndSwap");
    if (cell.value !== old) {
      return false;
    }
    cell.value = next;
    return true;
  }

  static Load(receiver: Uint64 | undefined): uint64 {
    return requireCell(receiver, "Uint64.Load").value;
  }
}

function requireCell<T>(cell: T | undefined, name: string): T {
  if (cell === undefined) {
    GoPanic.raiseRuntime(`${name} called with nil receiver`);
  }
  return cell;
}

function toInt32(value: number): int32 {
  return value | 0;
}

function toUint32(value: number): uint32 {
  return value >>> 0;
}
