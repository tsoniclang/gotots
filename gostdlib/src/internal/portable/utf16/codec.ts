import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int32, int64, uint16 } from "@gotots/runtime/scalars.js";

import { RuneError, validRune } from "../utf8/codec.js";

const surrogateMin = 0xd800;
const surrogateMax = 0xdfff;
const highSurrogateMax = 0xdbff;

export function IsSurrogate(rune: int32): boolean {
  return rune >= surrogateMin && rune <= surrogateMax;
}

export function DecodeRune(high: int32, low: int32): int32 {
  if (high >= surrogateMin && high <= highSurrogateMax && low >= 0xdc00 && low <= surrogateMax) {
    return ((high - surrogateMin) * 0x400) + (low - 0xdc00) + 0x10000;
  }
  return RuneError;
}

export function EncodeRune(rune: int32): [int32, int32] {
  if (rune < 0x10000 || rune > 0x10ffff) {
    return [RuneError, RuneError];
  }
  const adjusted = rune - 0x10000;
  return [surrogateMin + (adjusted >> 10), 0xdc00 + (adjusted & 0x3ff)];
}

export function RuneLen(rune: int32): int64 {
  if (!validRune(rune) || IsSurrogate(rune)) {
    return -1;
  }
  return rune < 0x10000 ? 1 : 2;
}

export function Decode(source: RuntimeSlice<uint16>): RuntimeSlice<int32> {
  const result: int32[] = [];
  for (let index = 0; index < source.length; index += 1) {
    const first = source.get(index);
    if (first < surrogateMin || first > surrogateMax) {
      result.push(first);
      continue;
    }
    if (first <= highSurrogateMax && index + 1 < source.length) {
      const second = source.get(index + 1);
      const decoded = DecodeRune(first, second);
      if (decoded !== RuneError) {
        result.push(decoded);
        index += 1;
        continue;
      }
    }
    result.push(RuneError);
  }
  return RuntimeSlice.literal(result);
}
