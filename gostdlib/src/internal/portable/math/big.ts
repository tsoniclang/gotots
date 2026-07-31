import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  bool,
  float64,
  gostring,
  int64,
  int8,
  uint64,
} from "@gotots/runtime/scalars.js";

const accuracyBelow: int8 = -1;
const accuracyExact: int8 = 0;
const accuracyAbove: int8 = 1;
const intValues = new WeakMap<Int, bigint>();
const floatValues = new WeakMap<Float, FloatState>();

interface FloatState {
  value: bigint;
  precision: number;
  accuracy: int8;
}

export class Accuracy {
  constructor(readonly value: int8) {}
}

export class Int {
  constructor(value = 0n) {
    intValues.set(this, value);
  }

  static Exp(
    receiver: Int | undefined,
    base: Int | undefined,
    exponent: Int | undefined,
    modulus: Int | undefined,
  ): Int | undefined {
    const target = requireInt(receiver);
    const sourceBase = intValue(requireInt(base));
    const sourceExponent = intValue(requireInt(exponent));
    const sourceModulus = modulus === undefined ? 0n : intValue(modulus);
    if (sourceExponent < 0n && sourceModulus !== 0n) {
      const inverse = modularInverse(sourceBase, absolute(sourceModulus));
      if (inverse === undefined) {
        return undefined;
      }
      intValues.set(
        target,
        modularPower(inverse, -sourceExponent, absolute(sourceModulus)),
      );
      return target;
    }
    if (sourceExponent <= 0n) {
      intValues.set(target, 1n);
      return target;
    }
    if (sourceModulus === 0n) {
      intValues.set(target, sourceBase ** sourceExponent);
      return target;
    }
    intValues.set(
      target,
      modularPower(sourceBase, sourceExponent, absolute(sourceModulus)),
    );
    return target;
  }

  static Float64(receiver: Int | undefined): [float64, Accuracy] {
    return integerFloat64(intValue(requireInt(receiver)));
  }

  static SetString(
    receiver: Int | undefined,
    text: gostring,
    base: int64,
  ): [Int | undefined, bool] {
    const target = requireInt(receiver);
    const parsed = parseInteger(text, base);
    if (parsed === undefined) {
      return [undefined, false];
    }
    intValues.set(target, parsed);
    return [target, true];
  }

  static String(receiver: Int | undefined): gostring {
    return intValue(requireInt(receiver)).toString(10);
  }
}

export class Float {
  constructor() {
    floatValues.set(this, {
      value: 0n,
      precision: 0,
      accuracy: accuracyExact,
    });
  }

  static Float64(receiver: Float | undefined): [float64, Accuracy] {
    const target = requireFloat(receiver);
    const [value, accuracy] = integerFloat64(floatState(target).value);
    return [value, accuracy];
  }

  static SetInt(receiver: Float | undefined, source: Int | undefined): Float | undefined {
    const target = requireFloat(receiver);
    const state = floatState(target);
    state.value = intValue(requireInt(source));
    if (state.precision === 0) {
      state.precision = Math.max(bitLength(state.value), 64);
      state.accuracy = accuracyExact;
      return target;
    }
    roundToPrecision(state);
    return target;
  }

  static SetPrec(receiver: Float | undefined, precision: uint64): Float | undefined {
    const target = requireFloat(receiver);
    const state = floatState(target);
    const selected = Math.max(0, Math.min(Math.trunc(precision), 4_294_967_295));
    state.precision = selected;
    if (selected === 0) {
      state.accuracy = state.value < 0n
        ? accuracyAbove
        : state.value > 0n
          ? accuracyBelow
          : accuracyExact;
      state.value = 0n;
      return target;
    }
    roundToPrecision(state);
    return target;
  }
}

export function NewInt(value: int64): Int | undefined {
  return new Int(BigInt(Math.trunc(value)));
}

function requireInt(value: Int | undefined): Int {
  if (value === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return value;
}

function requireFloat(value: Float | undefined): Float {
  if (value === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return value;
}

function intValue(value: Int): bigint {
  const result = intValues.get(value);
  if (result === undefined) {
    GoPanic.raiseRuntime("invalid big.Int state");
  }
  return result;
}

function floatState(value: Float): FloatState {
  const result = floatValues.get(value);
  if (result === undefined) {
    GoPanic.raiseRuntime("invalid big.Float state");
  }
  return result;
}

function absolute(value: bigint): bigint {
  return value < 0n ? -value : value;
}

function bitLength(value: bigint): number {
  return absolute(value).toString(2).length;
}

function roundToPrecision(target: FloatState): void {
  const magnitude = absolute(target.value);
  const length = bitLength(magnitude);
  if (magnitude === 0n || length <= target.precision) {
    target.accuracy = accuracyExact;
    return;
  }
  const shift = BigInt(length - target.precision);
  const unit = 1n << shift;
  const quotient = magnitude / unit;
  const remainder = magnitude % unit;
  const half = unit >> 1n;
  const increment = remainder > half || (remainder === half && (quotient & 1n) === 1n);
  const roundedMagnitude = (quotient + (increment ? 1n : 0n)) * unit;
  const rounded = target.value < 0n ? -roundedMagnitude : roundedMagnitude;
  target.accuracy = rounded < target.value ? accuracyBelow : accuracyAbove;
  target.value = rounded;
}

function integerFloat64(value: bigint): [float64, Accuracy] {
  const result = Number(value);
  if (!Number.isFinite(result)) {
    return [result, new Accuracy(value < 0n ? accuracyBelow : accuracyAbove)];
  }
  const represented = BigInt(result);
  if (represented === value) {
    return [result, new Accuracy(accuracyExact)];
  }
  return [
    result,
    new Accuracy(represented < value ? accuracyBelow : accuracyAbove),
  ];
}

function modularPower(base: bigint, exponent: bigint, modulus: bigint): bigint {
  if (modulus === 1n) {
    return 0n;
  }
  let factor = ((base % modulus) + modulus) % modulus;
  let power = exponent;
  let result = 1n;
  while (power > 0n) {
    if ((power & 1n) === 1n) {
      result = (result * factor) % modulus;
    }
    factor = (factor * factor) % modulus;
    power >>= 1n;
  }
  return result;
}

function modularInverse(value: bigint, modulus: bigint): bigint | undefined {
  let oldRemainder = ((value % modulus) + modulus) % modulus;
  let remainder = modulus;
  let oldCoefficient = 1n;
  let coefficient = 0n;
  while (remainder !== 0n) {
    const quotient = oldRemainder / remainder;
    [oldRemainder, remainder] = [remainder, oldRemainder - quotient * remainder];
    [oldCoefficient, coefficient] = [coefficient, oldCoefficient - quotient * coefficient];
  }
  if (oldRemainder !== 1n) {
    return undefined;
  }
  return ((oldCoefficient % modulus) + modulus) % modulus;
}

function parseInteger(text: gostring, requestedBase: int64): bigint | undefined {
  if (requestedBase !== 0 && (requestedBase < 2 || requestedBase > 62)) {
    GoPanic.raiseRuntime("invalid number base");
  }
  if (text.length === 0 || text.trim() !== text) {
    return undefined;
  }
  let index = 0;
  let negative = false;
  if (text[0] === "+" || text[0] === "-") {
    negative = text[0] === "-";
    index += 1;
  }
  if (index === text.length) {
    return undefined;
  }

  let base = requestedBase;
  let prefix = false;
  if (base === 0) {
    base = 10;
    if (text[index] === "0") {
      const marker = text[index + 1];
      if (marker === "b" || marker === "B") {
        base = 2;
        index += 2;
        prefix = true;
      } else if (marker === "o" || marker === "O") {
        base = 8;
        index += 2;
        prefix = true;
      } else if (marker === "x" || marker === "X") {
        base = 16;
        index += 2;
        prefix = true;
      } else {
        base = 8;
      }
    }
  }

  let sawDigit = false;
  let previousUnderscore = false;
  let result = 0n;
  for (; index < text.length; index += 1) {
    const character = text[index] ?? "";
    if (character === "_") {
      if (requestedBase !== 0 || previousUnderscore || (!sawDigit && !prefix)) {
        return undefined;
      }
      previousUnderscore = true;
      prefix = false;
      continue;
    }
    const digit = digitValue(character, base);
    if (digit < 0 || digit >= base) {
      return undefined;
    }
    result = result * BigInt(base) + BigInt(digit);
    sawDigit = true;
    previousUnderscore = false;
    prefix = false;
  }
  if (!sawDigit || previousUnderscore) {
    return undefined;
  }
  return negative ? -result : result;
}

function digitValue(character: string, base: number): number {
  const code = character.charCodeAt(0);
  if (code >= 48 && code <= 57) {
    return code - 48;
  }
  if (code >= 97 && code <= 122) {
    return code - 97 + 10;
  }
  if (code >= 65 && code <= 90) {
    return base <= 36 ? code - 65 + 10 : code - 65 + 36;
  }
  return -1;
}
