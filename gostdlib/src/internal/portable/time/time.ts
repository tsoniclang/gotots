import type { bool, gostring, int64 } from "@gotots/runtime/scalars.js";
import {
  monotonicMilliseconds,
  wallMilliseconds,
} from "../../node/time/clock.js";
import { Duration } from "./duration.js";

const nanosecondsPerMillisecond = 1_000_000;
const zeroEpochMilliseconds = -62_135_596_800_000;

let equalTimeRepresentation: (left: Time, right: Time) => bool;
let hashTimeRepresentation: (source: Time) => number;

export class Time {
  constructor(
    private readonly epochMilliseconds: number | undefined = undefined,
    private readonly monotonic: number | undefined = undefined,
    private readonly nanosecondRemainder: number = 0,
  ) {}

  static {
    equalTimeRepresentation = (left: Time, right: Time): bool => (
      left.epochMilliseconds === right.epochMilliseconds
        && left.monotonic === right.monotonic
        && left.nanosecondRemainder === right.nanosecondRemainder
    );
    hashTimeRepresentation = (source: Time): number => {
      let hash = 2_166_136_261;
      hash = hashTimeNumber(hash, source.epochMilliseconds);
      hash = hashTimeNumber(hash, source.monotonic);
      return hashTimeNumber(hash, source.nanosecondRemainder);
    };
  }

  Add(d: Duration): Time {
    if (this.epochMilliseconds === undefined) {
      return new Time();
    }
    const deltaNanoseconds = d.Nanoseconds();
    const remainderTotal = this.nanosecondRemainder + deltaNanoseconds;
    const deltaMilliseconds = Math.floor(
      remainderTotal / nanosecondsPerMillisecond,
    );
    const remainder = remainderTotal
      - deltaMilliseconds * nanosecondsPerMillisecond;
    return new Time(
      this.epochMilliseconds + deltaMilliseconds,
      this.monotonic === undefined
        ? undefined
        : this.monotonic + deltaNanoseconds / nanosecondsPerMillisecond,
      remainder,
    );
  }

  After(u: Time): bool {
    return Time.#compare(this, u) > 0;
  }

  Before(u: Time): bool {
    return Time.#compare(this, u) < 0;
  }

  Equal(u: Time): bool {
    return Time.#compare(this, u) === 0;
  }

  Format(layout: gostring): gostring {
    return Time.#format(this, layout);
  }

  Sub(u: Time): Duration {
    if (this.monotonic !== undefined && u.monotonic !== undefined) {
      return new Duration(
        Math.trunc((this.monotonic - u.monotonic) * nanosecondsPerMillisecond),
      );
    }
    return new Duration(Math.trunc((
      (this.epochMilliseconds ?? zeroEpochMilliseconds)
        - (u.epochMilliseconds ?? zeroEpochMilliseconds)
    ) * nanosecondsPerMillisecond
      + this.nanosecondRemainder
      - u.nanosecondRemainder));
  }

  IsZero(): bool {
    return this.epochMilliseconds === undefined;
  }

  String(): gostring {
    if (this.epochMilliseconds === undefined) {
      return "0001-01-01 00:00:00 +0000 UTC";
    }
    return Time.#format(this, "2006-01-02 15:04:05.000000000 -0700 MST");
  }

  UnixMilli(): int64 {
    return Math.trunc(this.epochMilliseconds ?? zeroEpochMilliseconds);
  }

  UnixNano(): int64 {
    return Math.trunc(
      this.UnixMilli() * nanosecondsPerMillisecond
        + this.nanosecondRemainder,
    );
  }

  static #compare(left: Time, right: Time): number {
    if (left.monotonic !== undefined && right.monotonic !== undefined) {
      return left.monotonic - right.monotonic;
    }
    const millisecondDifference = (
      (left.epochMilliseconds ?? zeroEpochMilliseconds)
        - (right.epochMilliseconds ?? zeroEpochMilliseconds)
    );
    if (millisecondDifference !== 0) {
      return millisecondDifference;
    }
    return left.nanosecondRemainder - right.nanosecondRemainder;
  }

  static #format(value: Time, layout: string): string {
    const zero = value.epochMilliseconds === undefined;
    const date = new Date(value.epochMilliseconds ?? 0);
    if (zero) {
      date.setUTCFullYear(1, 0, 1);
      date.setUTCHours(0, 0, 0, 0);
    }
    const year = zero ? date.getUTCFullYear() : date.getFullYear();
    const month = zero ? date.getUTCMonth() : date.getMonth();
    const day = zero ? date.getUTCDate() : date.getDate();
    const weekday = zero ? date.getUTCDay() : date.getDay();
    const hour24 = zero ? date.getUTCHours() : date.getHours();
    const minute = zero ? date.getUTCMinutes() : date.getMinutes();
    const second = zero ? date.getUTCSeconds() : date.getSeconds();
    const milliseconds = zero ? date.getUTCMilliseconds() : date.getMilliseconds();
    const hour12 = hour24 % 12 || 12;
    const nanoseconds = String(
      milliseconds * nanosecondsPerMillisecond
        + value.nanosecondRemainder,
    ).padStart(9, "0");
    const offsetMinutes = zero ? 0 : -date.getTimezoneOffset();
    const offsetSign = offsetMinutes < 0 ? "-" : "+";
    const offsetHours = pad(Math.floor(Math.abs(offsetMinutes) / 60));
    const offsetRemainder = pad(Math.abs(offsetMinutes) % 60);
    const offsetColon = `${offsetSign}${offsetHours}:${offsetRemainder}`;
    const offsetCompact = `${offsetSign}${offsetHours}${offsetRemainder}`;
    const offsetWithSeconds = `${offsetColon}:00`;
    const offsetCompactSeconds = `${offsetCompact}00`;
    const zone = offsetMinutes === 0 ? "UTC" : `GMT${offsetColon}`;
    const replacements: Readonly<Record<string, string>> = {
      "2006": String(year).padStart(4, "0"),
      "06": String(year % 100).padStart(2, "0"),
      "January": monthNames[month] ?? "",
      "Jan": monthNames[month]?.slice(0, 3) ?? "",
      "01": pad(month + 1),
      "1": String(month + 1),
      "Monday": dayNames[weekday] ?? "",
      "Mon": dayNames[weekday]?.slice(0, 3) ?? "",
      "002": String(dayOfYear(year, month, day)).padStart(3, "0"),
      "__2": String(dayOfYear(year, month, day)).padStart(3, " "),
      "02": pad(day),
      "_2": String(day).padStart(2, " "),
      "2": String(day),
      "15": pad(hour24),
      "03": pad(hour12),
      "3": String(hour12),
      "04": pad(minute),
      "4": String(minute),
      "05": pad(second),
      "5": String(second),
      "PM": hour24 < 12 ? "AM" : "PM",
      "pm": hour24 < 12 ? "am" : "pm",
      "MST": zone,
      "Z07:00:00": offsetMinutes === 0 ? "Z" : offsetWithSeconds,
      "-07:00:00": offsetWithSeconds,
      "Z070000": offsetMinutes === 0 ? "Z" : offsetCompactSeconds,
      "-070000": offsetCompactSeconds,
      "Z07:00": offsetMinutes === 0 ? "Z" : offsetColon,
      "-07:00": offsetColon,
      "Z0700": offsetMinutes === 0 ? "Z" : offsetCompact,
      "-0700": offsetCompact,
      "Z07": offsetMinutes === 0 ? "Z" : `${offsetSign}${offsetHours}`,
      "-07": `${offsetSign}${offsetHours}`,
    };
    return layout.replace(
      /(?:[.,](?:0{1,9}|9{1,9})|Z07:00:00|-07:00:00|Z070000|-070000|Z07:00|-07:00|January|Monday|Z0700|-0700|2006|__2|002|Jan|Mon|MST|Z07|-07|15|03|04|05|01|02|_2|PM|pm|06|1|2|3|4|5)/gu,
      (token) => {
        const fractionDigits = token.slice(1);
        if ((token[0] === "." || token[0] === ",")
          && /^(?:0{1,9}|9{1,9})$/u.test(fractionDigits)) {
          const digits = nanoseconds.slice(0, fractionDigits.length);
          if (fractionDigits[0] === "9") {
            const trimmed = digits.replace(/0+$/u, "");
            return trimmed === "" ? "" : `${token[0]}${trimmed}`;
          }
          return `${token[0]}${digits}`;
        }
        return replacements[token] ?? token;
      },
    );
  }
}

export function timeRepresentationEqual(left: Time, right: Time): bool {
  return equalTimeRepresentation(left, right);
}

export function timeRepresentationHash(source: Time): number {
  return hashTimeRepresentation(source);
}

export function Now(): Time {
  return new Time(wallMilliseconds(), monotonicMilliseconds());
}

export function Since(t: Time): Duration {
  return Now().Sub(t);
}

export function Unix(sec: int64, nsec: int64): Time {
  let seconds = sec;
  let nanoseconds = nsec;
  if (nanoseconds < 0 || nanoseconds >= 1_000_000_000) {
    const secondAdjustment = Math.trunc(nanoseconds / 1_000_000_000);
    seconds += secondAdjustment;
    nanoseconds -= secondAdjustment * 1_000_000_000;
    if (nanoseconds < 0) {
      nanoseconds += 1_000_000_000;
      seconds -= 1;
    }
  }
  const milliseconds = seconds * 1_000
    + Math.floor(nanoseconds / nanosecondsPerMillisecond);
  const remainder = nanoseconds % nanosecondsPerMillisecond;
  return new Time(milliseconds, undefined, remainder);
}

export function UnixMilli(msec: int64): Time {
  return new Time(msec);
}

export function Until(t: Time): Duration {
  return t.Sub(Now());
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

function hashTimeNumber(hash: number, value: number | undefined): number {
  if (value === undefined) {
    return Math.imul(hash ^ 0x9e37_79b9, 16_777_619) >>> 0;
  }
  const text = String(value);
  let result = hash;
  for (let index = 0; index < text.length; index += 1) {
    result = Math.imul(result ^ text.charCodeAt(index), 16_777_619) >>> 0;
  }
  return Math.imul(result ^ 0xff, 16_777_619) >>> 0;
}

function dayOfYear(year: number, month: number, day: number): number {
  const monthLengths = [
    31,
    isLeapYear(year) ? 29 : 28,
    31, 30, 31, 30, 31, 31, 30, 31, 30, 31,
  ] as const;
  let result = day;
  for (let index = 0; index < month; index += 1) {
    result += monthLengths[index] ?? 0;
  }
  return result;
}

function isLeapYear(year: number): boolean {
  return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
}

const monthNames = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
] as const;

const dayNames = [
  "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday",
] as const;
