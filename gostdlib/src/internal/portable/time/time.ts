import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int64,
  uint8,
} from "@gotots/runtime/scalars.js";
import {
  monotonicMilliseconds,
  wallMilliseconds,
} from "../../node/time/clock.js";
import { ProviderError } from "../../runtime/error.js";
import { Duration } from "./duration.js";
import { parseRFC3339Bytes } from "./rfc3339.js";

const nanosecondsPerMillisecond = 1_000_000;
const zeroEpochMilliseconds = -62_135_596_800_000;
const rfc3339Nano = "2006-01-02T15:04:05.999999999Z07:00";

let equalTimeRepresentation: (left: Time, right: Time) => bool;
let hashTimeRepresentation: (source: Time) => number;
let assignTimeRepresentation: (target: Time, source: Time) => void;
let copyTimeRepresentation: (source: Time) => Time;

export class Time {
  constructor(
    private epochMilliseconds: number | undefined = undefined,
    private monotonic: number | undefined = undefined,
    private nanosecondRemainder: number = 0,
    private offsetSeconds: number | undefined = undefined,
    private zoneName: string | undefined = undefined,
  ) {}

  static {
    assignTimeRepresentation = (target: Time, source: Time): void => {
      target.epochMilliseconds = source.epochMilliseconds;
      target.monotonic = source.monotonic;
      target.nanosecondRemainder = source.nanosecondRemainder;
      target.offsetSeconds = source.offsetSeconds;
      target.zoneName = source.zoneName;
    };
    copyTimeRepresentation = (source: Time): Time => new Time(
      source.epochMilliseconds,
      source.monotonic,
      source.nanosecondRemainder,
      source.offsetSeconds,
      source.zoneName,
    );
    equalTimeRepresentation = (left: Time, right: Time): bool => (
      left.epochMilliseconds === right.epochMilliseconds
        && left.monotonic === right.monotonic
        && left.nanosecondRemainder === right.nanosecondRemainder
        && left.offsetSeconds === right.offsetSeconds
        && left.zoneName === right.zoneName
    );
    hashTimeRepresentation = (source: Time): number => {
      let hash = 2_166_136_261;
      hash = hashTimeNumber(hash, source.epochMilliseconds);
      hash = hashTimeNumber(hash, source.monotonic);
      hash = hashTimeNumber(hash, source.nanosecondRemainder);
      hash = hashTimeNumber(hash, source.offsetSeconds);
      return hashTimeString(hash, source.zoneName);
    };
  }

  static UnmarshalText(
    receiver: Time | undefined,
    source: RuntimeSlice<uint8>,
  ): GoError | undefined {
    if (receiver === undefined) {
      return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    return receiver.UnmarshalText(source);
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
      this.offsetSeconds,
      this.zoneName,
    );
  }

  AppendFormat(
    target: RuntimeSlice<uint8>,
    layout: gostring,
  ): RuntimeSlice<uint8> {
    return target.append(0, Array.from(new TextEncoder().encode(this.Format(layout))));
  }

  AppendText(
    target: RuntimeSlice<uint8>,
  ): [RuntimeSlice<uint8>, GoError | undefined] {
    const [text, failure] = this.#rfc3339Text("AppendText");
    if (failure !== undefined) {
      return [RuntimeSlice.nil<uint8>(), failure];
    }
    return [
      target.append(0, Array.from(new TextEncoder().encode(text))),
      undefined,
    ];
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

  MarshalText(): [RuntimeSlice<uint8>, GoError | undefined] {
    const [text, failure] = this.#rfc3339Text("MarshalText");
    if (failure !== undefined) {
      return [RuntimeSlice.nil<uint8>(), failure];
    }
    return [
      RuntimeSlice.literal(Array.from(new TextEncoder().encode(text))),
      undefined,
    ];
  }

  MarshalJSON(): [RuntimeSlice<uint8>, GoError | undefined] {
    const [text, failure] = this.#rfc3339Text("MarshalJSON");
    if (failure !== undefined) {
      return [RuntimeSlice.nil<uint8>(), failure];
    }
    return [
      RuntimeSlice.literal(Array.from(
        new TextEncoder().encode(JSON.stringify(text)),
      )),
      undefined,
    ];
  }

  String(): gostring {
    if (this.epochMilliseconds === undefined) {
      return "0001-01-01 00:00:00 +0000 UTC";
    }
    return Time.#format(this, "2006-01-02 15:04:05.000000000 -0700 MST");
  }

  Nanosecond(): number {
    const millisecond = (
      (this.epochMilliseconds ?? zeroEpochMilliseconds) % 1_000 + 1_000
    ) % 1_000;
    return millisecond * nanosecondsPerMillisecond
      + this.nanosecondRemainder;
  }

  Unix(): int64 {
    return Math.floor(
      (this.epochMilliseconds ?? zeroEpochMilliseconds) / 1_000,
    );
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

  UnmarshalText(source: RuntimeSlice<uint8>): GoError | undefined {
    const parsed = parseRFC3339Bytes(source);
    if (!parsed.ok) {
      this.epochMilliseconds = undefined;
      this.monotonic = undefined;
      this.nanosecondRemainder = 0;
      this.offsetSeconds = undefined;
      this.zoneName = undefined;
      return parsed.failure;
    }
    this.epochMilliseconds = parsed.epochMilliseconds;
    this.monotonic = undefined;
    this.nanosecondRemainder = parsed.nanosecondRemainder;
    this.offsetSeconds = parsed.offsetSeconds;
    this.zoneName = parsed.zoneName;
    return undefined;
  }

  UTC(): Time {
    if (this.epochMilliseconds === undefined) {
      return new Time();
    }
    return new Time(
      this.epochMilliseconds,
      undefined,
      this.nanosecondRemainder,
      0,
      "UTC",
    );
  }

  #rfc3339Text(
    method: "AppendText" | "MarshalJSON" | "MarshalText",
  ): [gostring, GoError | undefined] {
    const text = Time.#format(this, rfc3339Nano);
    if (!/^\d{4}-/u.test(text)) {
      return [
        "",
        new ProviderError(`Time.${method}: year outside of range [0,9999]`),
      ];
    }
    return [text, undefined];
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
    const fixedOffset = value.offsetSeconds;
    const fixedLocation = fixedOffset !== undefined;
    const date = new Date(
      (value.epochMilliseconds ?? 0)
        + (fixedOffset ?? 0) * 1_000,
    );
    if (zero) {
      date.setUTCFullYear(1, 0, 1);
      date.setUTCHours(0, 0, 0, 0);
    }
    const utcComponents = zero || fixedLocation;
    const year = utcComponents ? date.getUTCFullYear() : date.getFullYear();
    const month = utcComponents ? date.getUTCMonth() : date.getMonth();
    const day = utcComponents ? date.getUTCDate() : date.getDate();
    const weekday = utcComponents ? date.getUTCDay() : date.getDay();
    const hour24 = utcComponents ? date.getUTCHours() : date.getHours();
    const minute = utcComponents ? date.getUTCMinutes() : date.getMinutes();
    const second = utcComponents ? date.getUTCSeconds() : date.getSeconds();
    const milliseconds = utcComponents
      ? date.getUTCMilliseconds()
      : date.getMilliseconds();
    const hour12 = hour24 % 12 || 12;
    const nanoseconds = String(
      milliseconds * nanosecondsPerMillisecond
        + value.nanosecondRemainder,
    ).padStart(9, "0");
    const offsetSeconds = zero
      ? 0
      : fixedOffset ?? -date.getTimezoneOffset() * 60;
    const offsetSign = offsetSeconds < 0 ? "-" : "+";
    const absoluteOffset = Math.abs(offsetSeconds);
    const offsetHours = pad(Math.floor(absoluteOffset / 3_600));
    const offsetRemainder = pad(Math.floor(absoluteOffset / 60) % 60);
    const offsetSecondRemainder = pad(absoluteOffset % 60);
    const offsetColon = `${offsetSign}${offsetHours}:${offsetRemainder}`;
    const offsetCompact = `${offsetSign}${offsetHours}${offsetRemainder}`;
    const offsetWithSeconds = `${offsetColon}:${offsetSecondRemainder}`;
    const offsetCompactSeconds = `${offsetCompact}${offsetSecondRemainder}`;
    const zone = value.zoneName
      ?? (offsetSeconds === 0 ? "UTC" : `GMT${offsetColon}`);
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
      "Z07:00:00": offsetSeconds === 0 ? "Z" : offsetWithSeconds,
      "-07:00:00": offsetWithSeconds,
      "Z070000": offsetSeconds === 0 ? "Z" : offsetCompactSeconds,
      "-070000": offsetCompactSeconds,
      "Z07:00": offsetSeconds === 0 ? "Z" : offsetColon,
      "-07:00": offsetColon,
      "Z0700": offsetSeconds === 0 ? "Z" : offsetCompact,
      "-0700": offsetCompact,
      "Z07": offsetSeconds === 0 ? "Z" : `${offsetSign}${offsetHours}`,
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

export function timeRepresentationAssign(target: Time, source: Time): void {
  assignTimeRepresentation(target, source);
}

export function timeRepresentationCopy(source: Time): Time {
  return copyTimeRepresentation(source);
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

function hashTimeString(hash: number, value: string | undefined): number {
  if (value === undefined) {
    return Math.imul(hash ^ 0x85eb_ca6b, 16_777_619) >>> 0;
  }
  let result = hash;
  for (let index = 0; index < value.length; index += 1) {
    result = Math.imul(result ^ value.charCodeAt(index), 16_777_619) >>> 0;
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
