import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int32, int64, uint16, uint32 } from "@gotots/runtime/scalars.js";

export class Range16 {
  constructor(
    public Lo: uint16 = 0,
    public Hi: uint16 = 0,
    public Stride: uint16 = 0,
  ) {}
}

export class Range32 {
  constructor(
    public Lo: uint32 = 0,
    public Hi: uint32 = 0,
    public Stride: uint32 = 0,
  ) {}
}

export class RangeTable {
  constructor(
    public R16: RuntimeSlice<Range16> = RuntimeSlice.nil<Range16>(),
    public R32: RuntimeSlice<Range32> = RuntimeSlice.nil<Range32>(),
    public LatinOffset: int64 = 0,
  ) {}
}

export function Is(table: RangeTable | undefined, rune: int32): boolean {
  if (table === undefined) {
    GoPanic.raiseRuntime("nil *unicode.RangeTable");
  }
  if (table.R16.length > 0) {
    const last = table.R16.get(table.R16.length - 1);
    if (rune >= 0 && rune <= last.Hi && inRanges16(table.R16, rune, 0)) {
      return true;
    }
  }
  if (table.R32.length > 0 && rune >= table.R32.get(0).Lo) {
    return inRanges32(table.R32, rune);
  }
  return false;
}

export function rangeTable(
  ranges16: readonly (readonly [number, number, number])[],
  ranges32: readonly (readonly [number, number, number])[],
  latinOffset: number,
): RangeTable {
  return new RangeTable(
    RuntimeSlice.literal(ranges16.map(([lo, hi, stride]) => new Range16(lo, hi, stride))),
    RuntimeSlice.literal(ranges32.map(([lo, hi, stride]) => new Range32(lo, hi, stride))),
    latinOffset,
  );
}

function inRanges16(ranges: RuntimeSlice<Range16>, rune: int32, start: number): boolean {
  let low = start;
  let high = ranges.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const range = ranges.get(middle);
    if (rune < range.Lo) {
      high = middle;
    } else if (rune > range.Hi) {
      low = middle + 1;
    } else {
      if (range.Stride === 0) {
        GoPanic.raiseRuntime("unicode.Range16 has zero stride");
      }
      return range.Stride === 1 || (rune - range.Lo) % range.Stride === 0;
    }
  }
  return false;
}

function inRanges32(ranges: RuntimeSlice<Range32>, rune: int32): boolean {
  let low = 0;
  let high = ranges.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const range = ranges.get(middle);
    if (rune < range.Lo) {
      high = middle;
    } else if (rune > range.Hi) {
      low = middle + 1;
    } else {
      if (range.Stride === 0) {
        GoPanic.raiseRuntime("unicode.Range32 has zero stride");
      }
      return range.Stride === 1 || (rune - range.Lo) % range.Stride === 0;
    }
  }
  return false;
}
