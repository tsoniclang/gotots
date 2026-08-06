import type { float64, gostring, int64 } from "@gotots/gostdlib/internal/scalars.js";

const nanosecondsPerSecond = 1_000_000_000n;
const nanosecondsPerMinute = 60n * nanosecondsPerSecond;
const nanosecondsPerHour = 60n * nanosecondsPerMinute;

export class Duration {
  readonly #value: int64;

  constructor(value: int64) {
    this.#value = value;
  }

  Nanoseconds(): int64 {
    return this.#value;
  }

  Seconds(): float64 {
    return Number(this.#value) / Number(nanosecondsPerSecond);
  }

  String(): gostring {
    if (this.#value === 0n) {
      return "0s";
    }
    const negative = this.#value < 0n;
    let remaining = negative ? -this.#value : this.#value;
    const sign = negative ? "-" : "";

    if (remaining < 1_000n) {
      return `${sign}${remaining}ns`;
    }
    if (remaining < 1_000_000n) {
      return `${sign}${formatFraction(remaining, 1_000n, "µs")}`;
    }
    if (remaining < nanosecondsPerSecond) {
      return `${sign}${formatFraction(remaining, 1_000_000n, "ms")}`;
    }

    const hours = remaining / nanosecondsPerHour;
    remaining -= hours * nanosecondsPerHour;
    const minutes = remaining / nanosecondsPerMinute;
    remaining -= minutes * nanosecondsPerMinute;
    const seconds = remaining / nanosecondsPerSecond;
    const fraction = remaining - seconds * nanosecondsPerSecond;
    const prefix = hours > 0n
      ? `${hours}h${minutes}m`
      : minutes > 0n
        ? `${minutes}m`
        : "";
    const decimal = fraction === 0n
      ? `${seconds}`
      : `${seconds}.${String(fraction).padStart(9, "0").replace(/0+$/u, "")}`;
    return `${sign}${prefix}${decimal}s`;
  }
}

function formatFraction(
  nanoseconds: bigint,
  unit: bigint,
  suffix: string,
): string {
  const whole = nanoseconds / unit;
  const remainder = nanoseconds - whole * unit;
  if (remainder === 0n) {
    return `${whole}${suffix}`;
  }
  const width = String(unit).length - 1;
  const fraction = String(remainder).padStart(width, "0").replace(/0+$/u, "");
  return `${whole}.${fraction}${suffix}`;
}
