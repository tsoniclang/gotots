import { GoPanic } from "@gotots/runtime/panic.js";
import type { int64, uint32, uint64 } from "@gotots/runtime/scalars.js";

const uint32Width = 32;
const uint64Width = 64;
const uint64WidthBigInt = 64n;

export function Add64(
  left: uint64,
  right: uint64,
  carry: uint64,
): [uint64, uint64] {
  const sum = uint64BigInt(left) + uint64BigInt(right) + uint64BigInt(carry);
  return [uint64Number(sum), Number(sum >> uint64WidthBigInt)];
}

export function Div64(
  high: uint64,
  low: uint64,
  divisor: uint64,
): [uint64, uint64] {
  const highBits = uint64BigInt(high);
  const lowBits = uint64BigInt(low);
  const divisorBits = uint64BigInt(divisor);
  if (divisorBits === 0n) {
    GoPanic.raiseRuntime("integer divide by zero");
  }
  if (divisorBits <= highBits) {
    GoPanic.raiseRuntime("integer overflow");
  }
  const dividend = (highBits << uint64WidthBigInt) | lowBits;
  return [
    uint64Number(dividend / divisorBits),
    uint64Number(dividend % divisorBits),
  ];
}

export function Len(value: uint64): int64 {
  const bits = uint64BigInt(value);
  return bits === 0n ? 0 : bits.toString(2).length;
}

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

export function OnesCount64(value: uint64): int64 {
  return OnesCount(value);
}

export function ReverseBytes32(value: uint32): uint32 {
  const source = value >>> 0;
  return (
    ((source & 0x0000_00ff) << 24) |
    ((source & 0x0000_ff00) << 8) |
    ((source & 0x00ff_0000) >>> 8) |
    ((source & 0xff00_0000) >>> 24)
  ) >>> 0;
}

export function ReverseBytes64(value: uint64): uint64 {
  let source = uint64BigInt(value);
  let result = 0n;
  for (let index = 0; index < 8; index += 1) {
    result = (result << 8n) | (source & 0xffn);
    source >>= 8n;
  }
  return uint64Number(result);
}

export function RotateLeft32(value: uint32, count: int64): uint32 {
  const shift = normalizedRotation(count, uint32Width);
  const source = value >>> 0;
  return ((source << shift) | (source >>> ((uint32Width - shift) % uint32Width))) >>> 0;
}

export function RotateLeft64(value: uint64, count: int64): uint64 {
  const shift = BigInt(normalizedRotation(count, uint64Width));
  const source = uint64BigInt(value);
  return uint64Number(
    (source << shift) |
      (source >> ((uint64WidthBigInt - shift) % uint64WidthBigInt)),
  );
}

export function Mul64(left: uint64, right: uint64): [uint64, uint64] {
  const product = BigInt.asUintN(128, uint64BigInt(left) * uint64BigInt(right));
  return [
    uint64Number(product >> uint64WidthBigInt),
    uint64Number(product),
  ];
}

function uint64BigInt(value: uint64): bigint {
  return BigInt.asUintN(uint64Width, BigInt(Math.trunc(value)));
}

function uint64Number(value: bigint): uint64 {
  return Number(BigInt.asUintN(uint64Width, value));
}

function normalizedRotation(value: int64, width: number): number {
  const remainder = Math.trunc(value) % width;
  return remainder < 0 ? remainder + width : remainder;
}
