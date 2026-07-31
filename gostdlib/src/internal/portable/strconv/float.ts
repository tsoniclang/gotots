import type { float64, gostring, int64 } from "@gotots/runtime/scalars.js";

import { NumberError, rangeError, syntaxError } from "./number-error.js";

const decimalPattern = /^[+-]?(?:(?:[0-9](?:_?[0-9])*)?(?:\.(?:[0-9](?:_?[0-9])*)?)|[0-9](?:_?[0-9])*)(?:[eE][+-]?[0-9](?:_?[0-9])*)?$/;
const hexadecimalPattern = /^([+-]?)0[xX]_?((?:[0-9a-fA-F](?:_?[0-9a-fA-F])*)?)(?:\.((?:[0-9a-fA-F](?:_?[0-9a-fA-F])*)?))?[pP]([+-]?[0-9](?:_?[0-9])*)$/;

export function ParseFloat(
  value: gostring,
  bitSize: int64,
): [float64, NumberError | undefined] {
  const lower = value.toLowerCase();
  if (lower === "nan") {
    return [Number.NaN, undefined];
  }
  if (lower === "inf" || lower === "+inf" || lower === "infinity" || lower === "+infinity") {
    return [Number.POSITIVE_INFINITY, undefined];
  }
  if (lower === "-inf" || lower === "-infinity") {
    return [Number.NEGATIVE_INFINITY, undefined];
  }

  const parsed = parseFinite(value);
  if (parsed === undefined) {
    return [0, syntaxError("ParseFloat", value)];
  }
  const rounded = bitSize === 32 ? Math.fround(parsed) : parsed;
  if (!Number.isFinite(rounded)) {
    return [rounded, rangeError("ParseFloat", value)];
  }
  return [rounded, undefined];
}

function parseFinite(value: gostring): number | undefined {
  if (decimalPattern.test(value)) {
    if (!/[0-9]/.test(value)) {
      return undefined;
    }
    const result = Number(value.replaceAll("_", ""));
    return Number.isNaN(result) ? undefined : result;
  }

  const match = hexadecimalPattern.exec(value);
  if (match === null) {
    return undefined;
  }
  const integer = (match[2] ?? "").replaceAll("_", "");
  const fraction = (match[3] ?? "").replaceAll("_", "");
  if (integer.length === 0 && fraction.length === 0) {
    return undefined;
  }
  let significand = 0;
  for (const character of integer) {
    significand = significand * 16 + hexadecimalDigit(character);
  }
  let divisor = 16;
  for (const character of fraction) {
    significand += hexadecimalDigit(character) / divisor;
    divisor *= 16;
  }
  const exponent = Number((match[4] ?? "0").replaceAll("_", ""));
  const sign = match[1] === "-" ? -1 : 1;
  return sign * significand * 2 ** exponent;
}

function hexadecimalDigit(character: string): number {
  const code = character.toLowerCase().charCodeAt(0);
  return code <= 0x39 ? code - 0x30 : code - 0x61 + 10;
}
