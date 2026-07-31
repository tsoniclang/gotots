import type {
  bool,
  float64,
  int32,
  int64,
  uint64,
} from "@gotots/runtime/scalars.js";

const floatBuffer = new ArrayBuffer(8);
const floatView = new DataView(floatBuffer);

export const MaxFloat64: float64 = Number.MAX_VALUE;
export const MaxInt: int64 = 9_223_372_036_854_775_807;
export const MaxInt32: int32 = 2_147_483_647;
export const MaxInt64: int64 = 9_223_372_036_854_775_807;
export const MinInt: int64 = -9_223_372_036_854_775_808;
export const MinInt64: int64 = -9_223_372_036_854_775_808;

export function Abs(value: float64): float64 {
  return Math.abs(value);
}

export function Ceil(value: float64): float64 {
  return Math.ceil(value);
}

export function Copysign(magnitude: float64, sign: float64): float64 {
  const absolute = Math.abs(magnitude);
  return sign < 0 || Object.is(sign, -0) ? -absolute : absolute;
}

export function Float64bits(value: float64): uint64 {
  floatView.setFloat64(0, value, false);
  return Number(floatView.getBigUint64(0, false));
}

export function Floor(value: float64): float64 {
  return Math.floor(value);
}

export function Inf(sign: int64): float64 {
  return sign >= 0 ? Number.POSITIVE_INFINITY : Number.NEGATIVE_INFINITY;
}

export function IsInf(value: float64, sign: int64): bool {
  if (sign > 0) {
    return value === Number.POSITIVE_INFINITY;
  }
  if (sign < 0) {
    return value === Number.NEGATIVE_INFINITY;
  }
  return value === Number.POSITIVE_INFINITY || value === Number.NEGATIVE_INFINITY;
}

export function IsNaN(value: float64): bool {
  return Number.isNaN(value);
}

export function Log2(value: float64): float64 {
  return Math.log2(value);
}

export function Min(left: float64, right: float64): float64 {
  return Math.min(left, right);
}

export function Mod(dividend: float64, divisor: float64): float64 {
  return dividend % divisor;
}

export function NaN(): float64 {
  return Number.NaN;
}

export function Pow(base: float64, exponent: float64): float64 {
  return Math.pow(base, exponent);
}

export function Trunc(value: float64): float64 {
  return Math.trunc(value);
}
