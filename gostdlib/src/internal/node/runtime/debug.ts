import { setFlagsFromString } from "node:v8";
import type { int64 } from "@gotots/gostdlib/internal/scalars.js";
import { hostInteger } from "../../host-integer.js";
import { stackBytes } from "./process.js";

let maximumStackBytes: int64 = 1_000_000_000n;

export function setMaximumStack(bytes: int64): int64 {
  const previous = maximumStackBytes;
  maximumStackBytes = bytes;
  setFlagsFromString(`--stack_size=${Math.max(0, Math.floor(hostInteger(bytes) / 1024))}`);
  return previous;
}

export function currentStack(): Uint8Array {
  return stackBytes();
}
