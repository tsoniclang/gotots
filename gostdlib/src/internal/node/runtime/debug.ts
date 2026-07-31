import { setFlagsFromString } from "node:v8";
import type { int64 } from "@gotots/runtime/scalars.js";
import { stackBytes } from "./process.js";

let maximumStackBytes: int64 = 1_000_000_000;

export function setMaximumStack(bytes: int64): int64 {
  const previous = maximumStackBytes;
  maximumStackBytes = bytes;
  setFlagsFromString(`--stack_size=${Math.max(0, Math.floor(bytes / 1024))}`);
  return previous;
}

export function currentStack(): Uint8Array {
  return stackBytes();
}
