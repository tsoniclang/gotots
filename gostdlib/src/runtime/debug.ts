import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";
import {
  currentStack,
  setMaximumStack,
} from "../internal/node/runtime/debug.js";
import { byteSlice } from "../internal/runtime/slice.js";

export function SetMaxStack(bytes: int64): int64 {
  return setMaximumStack(bytes);
}

export function Stack(): RuntimeSlice<uint8> {
  return byteSlice(currentStack());
}
