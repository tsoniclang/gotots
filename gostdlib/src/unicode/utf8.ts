import type { gostring, int32, int64 } from "@gotots/runtime/scalars.js";

import {
  decodeLastRune,
  decodeRuneAt,
  RuneError as runeError,
  RuneSelf as runeSelf,
} from "../internal/portable/utf8/codec.js";

export const RuneError: int32 = runeError;
export const RuneSelf: int32 = runeSelf;

export function DecodeRuneInString(value: gostring): [int32, int64] {
  return decodeRuneAt(value, 0);
}

export function DecodeLastRuneInString(value: gostring): [int32, int64] {
  return decodeLastRune(value);
}
