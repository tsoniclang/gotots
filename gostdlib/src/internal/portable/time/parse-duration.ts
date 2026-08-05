import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";

import { ProviderError } from "../../runtime/error.js";
import { Duration } from "./duration.js";
import { timeQuote } from "./quote.js";

const maximumMagnitude = 1n << 63n;
const maximumPositive = maximumMagnitude - 1n;

const nanosecond = 1n;
const microsecond = 1_000n * nanosecond;
const millisecond = 1_000n * microsecond;
const second = 1_000n * millisecond;
const minute = 60n * second;
const hour = 60n * minute;

const parseDurationErrorType = Object.freeze({ comparable: true });

class ParseDurationError extends ProviderError {
  override readonly $go$type = parseDurationErrorType;

  constructor(message: string, value: string) {
    super(`time: ${message} ${timeQuote(value)}`);
  }

  override $go$format(
    verb: string,
    flags: string,
    precision: number | undefined,
  ): string {
    if (verb === "T") {
      return "*time.parseDurationError";
    }
    return super.$go$format(verb, flags, precision);
  }
}

type LeadingInteger = {
  readonly value: bigint;
  readonly next: number;
  readonly valid: boolean;
};

type LeadingFraction = {
  readonly value: bigint;
  readonly scale: bigint;
  readonly next: number;
};

export function ParseDuration(
  source: gostring,
): [Duration, GoError | undefined] {
  const original = source;
  let index = 0;
  let negative = false;

  const sign = source[index];
  if (sign === "-" || sign === "+") {
    negative = sign === "-";
    index += 1;
  }
  if (source.slice(index) === "0") {
    return [new Duration(0n), undefined];
  }
  if (index === source.length) {
    return failure("invalid duration", original);
  }

  let total = 0n;
  while (index < source.length) {
    const first = source[index];
    if (first !== "." && !isDigit(first)) {
      return failure("invalid duration", original);
    }

    const integerStart = index;
    const integer = leadingInteger(source, index);
    if (!integer.valid) {
      return failure("invalid duration", original);
    }
    index = integer.next;
    const hasInteger = index !== integerStart;

    let fraction = 0n;
    let scale = 1n;
    let hasFraction = false;
    if (source[index] === ".") {
      index += 1;
      const fractionStart = index;
      const selected = leadingFraction(source, index);
      fraction = selected.value;
      scale = selected.scale;
      index = selected.next;
      hasFraction = index !== fractionStart;
    }
    if (!hasInteger && !hasFraction) {
      return failure("invalid duration", original);
    }

    const unitStart = index;
    while (index < source.length) {
      const current = source[index];
      if (current === "." || isDigit(current)) {
        break;
      }
      index += 1;
    }
    if (index === unitStart) {
      return failure("missing unit in duration", original);
    }
    const unitText = source.slice(unitStart, index);
    const unit = durationUnit(unitText);
    if (unit === undefined) {
      return failure(
        `unknown unit ${timeQuote(unitText)} in duration`,
        original,
      );
    }

    if (integer.value > maximumMagnitude / unit) {
      return failure("invalid duration", original);
    }
    let value = integer.value * unit;
    if (fraction > 0n) {
      value += (fraction * unit) / scale;
      if (value > maximumMagnitude) {
        return failure("invalid duration", original);
      }
    }
    total += value;
    if (total > maximumMagnitude) {
      return failure("invalid duration", original);
    }
  }

  if (!negative && total > maximumPositive) {
    return failure("invalid duration", original);
  }
  const signed = negative ? -total : total;
  return [new Duration(signed), undefined];
}

function leadingInteger(source: string, start: number): LeadingInteger {
  let value = 0n;
  let index = start;
  while (index < source.length && isDigit(source[index])) {
    value = value * 10n + BigInt((source.charCodeAt(index) - 48));
    if (value > maximumMagnitude) {
      return { value: 0n, next: index, valid: false };
    }
    index += 1;
  }
  return { value, next: index, valid: true };
}

function leadingFraction(source: string, start: number): LeadingFraction {
  let value = 0n;
  let scale = 1n;
  let overflow = false;
  let index = start;
  while (index < source.length && isDigit(source[index])) {
    if (!overflow) {
      const next = value * 10n + BigInt(source.charCodeAt(index) - 48);
      if (value > maximumPositive / 10n || next > maximumMagnitude) {
        overflow = true;
      } else {
        value = next;
        scale *= 10n;
      }
    }
    index += 1;
  }
  return { value, scale, next: index };
}

function durationUnit(unit: string): bigint | undefined {
  switch (unit) {
    case "ns": return nanosecond;
    case "us":
    case "µs":
    case "μs": return microsecond;
    case "ms": return millisecond;
    case "s": return second;
    case "m": return minute;
    case "h": return hour;
    default: return undefined;
  }
}

function isDigit(value: string | undefined): boolean {
  return value !== undefined && value >= "0" && value <= "9";
}

function failure(message: string, source: string): [Duration, GoError] {
  return [new Duration(0n), new ParseDurationError(message, source)];
}
