import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint16, uint32 } from "@gotots/gostdlib/internal/scalars.js";

import { Range16, Range32, RangeTable } from "../../unicode.js";

export type UnicodeRangeTableStorage = RangeTable;

export class UnicodeRange16Operations {
  static $make(lo: uint16, hi: uint16, stride: uint16): Range16 {
    return new Range16(lo, hi, stride);
  }
}

export class UnicodeRange32Operations {
  static $make(lo: uint32, hi: uint32, stride: uint32): Range32 {
    return new Range32(lo, hi, stride);
  }
}

export class UnicodeRangeTableOperations {
  static $make(
    ranges16: RuntimeSlice<Range16>,
    ranges32: RuntimeSlice<Range32>,
    latinOffset: int64,
  ): RangeTable {
    return new RangeTable(ranges16, ranges32, latinOffset);
  }

  static $storageOf(source: RangeTable): UnicodeRangeTableStorage {
    return source;
  }

  static $fromStorage(source: UnicodeRangeTableStorage): RangeTable {
    return source;
  }
}
