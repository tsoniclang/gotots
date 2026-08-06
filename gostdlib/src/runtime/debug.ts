import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import {
  currentStack,
  setMaximumStack,
} from "../internal/node/runtime/debug.js";
import { byteSlice } from "../internal/runtime/slice.js";

export function SetMaxStack(bytes: int): int {
  return setMaximumStack(bytes);
}

export function Stack(): RuntimeSlice<uint8> {
  return byteSlice(currentStack());
}
