import type { float64, gostring, int64 } from "@gotots/runtime/scalars.js";

const nanosecondsPerSecond = 1_000_000_000;
const nanosecondsPerMinute = 60 * nanosecondsPerSecond;
const nanosecondsPerHour = 60 * nanosecondsPerMinute;

export class Duration {
  readonly #value: int64;

  constructor(value: int64) {
    this.#value = value;
  }

  Nanoseconds(): int64 {
    return this.#value;
  }

  Seconds(): float64 {
    return this.#value / nanosecondsPerSecond;
  }

  String(): gostring {
    if (this.#value === 0) {
      return "0s";
    }
    const negative = this.#value < 0;
    let remaining = Math.abs(Math.trunc(this.#value));
    const sign = negative ? "-" : "";

    if (remaining < 1_000) {
      return `${sign}${remaining}ns`;
    }
    if (remaining < 1_000_000) {
      return `${sign}${formatFraction(remaining, 1_000, "µs")}`;
    }
    if (remaining < nanosecondsPerSecond) {
      return `${sign}${formatFraction(remaining, 1_000_000, "ms")}`;
    }

    const hours = Math.floor(remaining / nanosecondsPerHour);
    remaining -= hours * nanosecondsPerHour;
    const minutes = Math.floor(remaining / nanosecondsPerMinute);
    remaining -= minutes * nanosecondsPerMinute;
    const seconds = Math.floor(remaining / nanosecondsPerSecond);
    const fraction = remaining - seconds * nanosecondsPerSecond;
    const prefix = hours > 0
      ? `${hours}h${minutes}m`
      : minutes > 0
        ? `${minutes}m`
        : "";
    const decimal = fraction === 0
      ? `${seconds}`
      : `${seconds}.${String(fraction).padStart(9, "0").replace(/0+$/u, "")}`;
    return `${sign}${prefix}${decimal}s`;
  }
}

function formatFraction(
  nanoseconds: number,
  unit: number,
  suffix: string,
): string {
  const whole = Math.floor(nanoseconds / unit);
  const remainder = nanoseconds - whole * unit;
  if (remainder === 0) {
    return `${whole}${suffix}`;
  }
  const width = Math.log10(unit);
  const fraction = String(remainder).padStart(width, "0").replace(/0+$/u, "");
  return `${whole}.${fraction}${suffix}`;
}
