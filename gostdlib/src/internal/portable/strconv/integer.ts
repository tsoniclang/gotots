import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  gostring,
  int,
  int64,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger } from "../../host-integer.js";

import { NumberError, rangeError, syntaxError } from "./number-error.js";

const digits = "0123456789abcdefghijklmnopqrstuvwxyz";

export function Atoi(value: gostring): [int, NumberError | undefined] {
  const [parsed, error] = ParseInt(value, 10n, 0n);
  return error === undefined
    ? [parsed, undefined]
    : [parsed, new NumberError("Atoi", value, error.Unwrap())];
}

export function FormatInt(value: int64, base: int): gostring {
  validateBase(base);
  const integer = BigInt.asIntN(64, value);
  return integer.toString(hostInteger(base));
}

export function FormatUint(value: uint64, base: int): gostring {
  validateBase(base);
  const integer = BigInt.asUintN(64, value);
  return integer.toString(hostInteger(base));
}

export function Itoa(value: int): gostring {
  return FormatInt(value, 10n);
}

export function ParseInt(
  value: gostring,
  base: int,
  bitSize: int,
): [int64, NumberError | undefined] {
  if (value.length === 0) {
    return [0n, syntaxError("ParseInt", value)];
  }
  let sign = 1n;
  let digitsText = value;
  if (digitsText[0] === "+" || digitsText[0] === "-") {
    sign = digitsText[0] === "-" ? -1n : 1n;
    digitsText = digitsText.slice(1);
  }
  if (digitsText.length === 0) {
    return [0n, syntaxError("ParseInt", value)];
  }

  const parsed = parseMagnitude(digitsText, base);
  if (parsed.error !== undefined) {
    return [0n, syntaxError("ParseInt", value)];
  }
  const bits = normalizedBits(bitSize);
  if (bits === undefined) {
    return [0n, syntaxError("ParseInt", value)];
  }
  const maximum = (1n << BigInt(bits - 1)) - 1n;
  const minimumMagnitude = 1n << BigInt(bits - 1);
  const magnitudeLimit = sign < 0n ? minimumMagnitude : maximum;
  if (parsed.value > magnitudeLimit) {
    return [sign < 0n ? -minimumMagnitude : maximum, rangeError("ParseInt", value)];
  }
  return [sign * parsed.value, undefined];
}

export function ParseUint(
  value: gostring,
  base: int,
  bitSize: int,
): [uint64, NumberError | undefined] {
  if (value.length === 0 || value[0] === "+" || value[0] === "-") {
    return [0n, syntaxError("ParseUint", value)];
  }
  const parsed = parseMagnitude(value, base);
  if (parsed.error !== undefined) {
    return [0n, syntaxError("ParseUint", value)];
  }
  const bits = normalizedBits(bitSize);
  if (bits === undefined) {
    return [0n, syntaxError("ParseUint", value)];
  }
  const maximum = (1n << BigInt(bits)) - 1n;
  if (parsed.value > maximum) {
    return [maximum, rangeError("ParseUint", value)];
  }
  return [parsed.value, undefined];
}

type Magnitude = {
  readonly value: bigint;
  readonly error: boolean | undefined;
};

function parseMagnitude(text: gostring, requestedBase: int): Magnitude {
  let base = hostInteger(requestedBase);
  let digitsText = text;
  let underscores = false;
  let prefixed = false;
  if (base === 0) {
    underscores = true;
    if (digitsText.startsWith("0x") || digitsText.startsWith("0X")) {
      base = 16;
      digitsText = digitsText.slice(2);
      prefixed = true;
    } else if (digitsText.startsWith("0b") || digitsText.startsWith("0B")) {
      base = 2;
      digitsText = digitsText.slice(2);
      prefixed = true;
    } else if (digitsText.startsWith("0o") || digitsText.startsWith("0O")) {
      base = 8;
      digitsText = digitsText.slice(2);
      prefixed = true;
    } else if (digitsText.startsWith("0") && digitsText.length > 1) {
      base = 8;
    } else {
      base = 10;
    }
  }
  if (!validBase(base) || digitsText.length === 0) {
    return { value: 0n, error: true };
  }

  let result = 0n;
  let sawDigit = false;
  let previousUnderscore = false;
  for (let index = 0; index < digitsText.length; index += 1) {
    const character = digitsText[index] ?? "";
    if (character === "_") {
      if (
        !underscores ||
        previousUnderscore ||
        (!sawDigit && !(prefixed && index === 0)) ||
        index + 1 === digitsText.length
      ) {
        return { value: 0n, error: true };
      }
      previousUnderscore = true;
      continue;
    }
    const digit = digitValue(character);
    if (digit < 0 || digit >= base) {
      return { value: 0n, error: true };
    }
    result = result * BigInt(base) + BigInt(digit);
    sawDigit = true;
    previousUnderscore = false;
  }
  return { value: result, error: sawDigit ? undefined : true };
}

function digitValue(character: string): number {
  return digits.indexOf(character.toLowerCase());
}

function normalizedBits(bitSize: int): number | undefined {
  if (bitSize < 0n || bitSize > 64n) {
    return undefined;
  }
  return bitSize === 0n ? 64 : hostInteger(bitSize);
}

function validateBase(base: int): void {
  if (!validBase(hostInteger(base))) {
    GoPanic.raiseRuntime("strconv: illegal base");
  }
}

function validBase(base: number): boolean {
  return Number.isInteger(base) && base >= 2 && base <= 36;
}
