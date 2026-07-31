import type { int64, uint32, uint64 } from "@gotots/runtime/scalars.js";

export function OnesCount(value: uint64): int64 {
  let remaining = BigInt.asUintN(64, BigInt(Math.trunc(value)));
  let count = 0;
  while (remaining !== 0n) {
    remaining &= remaining - 1n;
    count += 1;
  }
  return count;
}

export function OnesCount32(value: uint32): int64 {
  let remaining = value >>> 0;
  remaining -= (remaining >>> 1) & 0x5555_5555;
  remaining = (remaining & 0x3333_3333) + ((remaining >>> 2) & 0x3333_3333);
  remaining = (remaining + (remaining >>> 4)) & 0x0f0f_0f0f;
  return (remaining * 0x0101_0101) >>> 24;
}
