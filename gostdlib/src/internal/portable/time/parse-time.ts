import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";

import { ParseError } from "./parse-error.js";
import { Time } from "./time.js";

type TokenKind =
  | "long-month"
  | "month"
  | "numeric-month"
  | "zero-month"
  | "long-weekday"
  | "weekday"
  | "day"
  | "space-day"
  | "zero-day"
  | "space-year-day"
  | "zero-year-day"
  | "hour"
  | "hour12"
  | "zero-hour12"
  | "minute"
  | "zero-minute"
  | "second"
  | "zero-second"
  | "year"
  | "short-year"
  | "upper-meridiem"
  | "lower-meridiem"
  | "zone-name"
  | "zone"
  | "iso-zone"
  | "fraction-zero"
  | "fraction-nine";

type LayoutPart = {
  readonly token: TokenKind | "literal";
  readonly text: string;
  readonly width: number;
};

type ParsedFields = {
  year: number;
  month: number;
  day: number;
  yearDay: number | undefined;
  hour: number;
  minute: number;
  second: number;
  nanosecond: number;
  offsetSeconds: number;
  zoneName: string | undefined;
  monthSet: boolean;
  daySet: boolean;
  morning: boolean;
  evening: boolean;
};

const shortMonths = [
  "jan", "feb", "mar", "apr", "may", "jun",
  "jul", "aug", "sep", "oct", "nov", "dec",
] as const;

const longMonths = [
  "january", "february", "march", "april", "may", "june",
  "july", "august", "september", "october", "november", "december",
] as const;

const shortWeekdays = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"] as const;
const longWeekdays = [
  "sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday",
] as const;

export function Parse(
  layout: gostring,
  source: gostring,
): [Time, GoError | undefined] {
  const parts = tokenizeLayout(layout);
  const expression = new RegExp(
    `^${parts.map((part, index) => patternFor(part, parts[index + 1])).join("")}$`,
    "u",
  );
  const captures = expression.exec(source);
  if (captures === null) {
    return rejected(layout, source, layout, source);
  }

  const fields: ParsedFields = {
    year: 0,
    month: 1,
    day: 1,
    yearDay: undefined,
    hour: 0,
    minute: 0,
    second: 0,
    nanosecond: 0,
    offsetSeconds: 0,
    zoneName: "UTC",
    monthSet: false,
    daySet: false,
    morning: false,
    evening: false,
  };

  let captureIndex = 1;
  for (const part of parts) {
    if (part.token === "literal") {
      continue;
    }
    const captured = captures[captureIndex];
    captureIndex += 1;
    if (captured === undefined) {
      continue;
    }
    if (!consume(fields, part, captured)) {
      return rejected(layout, source, part.text, captured);
    }
  }

  if (fields.evening && fields.hour < 12) {
    fields.hour += 12;
  } else if (fields.morning && fields.hour === 12) {
    fields.hour = 0;
  }
  if (!applyYearDay(fields)
    || fields.month < 1 || fields.month > 12
    || fields.day < 1 || fields.day > daysInMonth(fields.year, fields.month)
    || fields.hour < 0 || fields.hour > 23
    || fields.minute < 0 || fields.minute > 59
    || fields.second < 0 || fields.second > 59) {
    return rejected(layout, source, "", source);
  }

  const milliseconds = Math.floor(fields.nanosecond / 1_000_000);
  const date = new Date(0);
  date.setUTCFullYear(fields.year, fields.month - 1, fields.day);
  date.setUTCHours(fields.hour, fields.minute, fields.second, milliseconds);
  return [new Time(
    date.getTime() - fields.offsetSeconds * 1_000,
    undefined,
    fields.nanosecond - milliseconds * 1_000_000,
    fields.offsetSeconds,
    fields.zoneName,
  ), undefined];
}

function consume(fields: ParsedFields, part: LayoutPart, source: string): boolean {
  const numeric = Number(source.trim().split(/[.,]/u, 1)[0]);
  switch (part.token) {
    case "year": fields.year = numeric; return true;
    case "short-year":
      fields.year = numeric >= 69 ? 1_900 + numeric : 2_000 + numeric;
      return true;
    case "long-month":
      return setMonth(fields, nameIndex(longMonths, source) + 1);
    case "month":
      return setMonth(fields, nameIndex(shortMonths, source) + 1);
    case "numeric-month":
    case "zero-month": return setMonth(fields, numeric);
    case "long-weekday": return nameIndex(longWeekdays, source) >= 0;
    case "weekday": return nameIndex(shortWeekdays, source) >= 0;
    case "day":
    case "space-day":
    case "zero-day": fields.day = numeric; fields.daySet = true; return true;
    case "space-year-day":
    case "zero-year-day": fields.yearDay = numeric; return true;
    case "hour": fields.hour = numeric; return numeric < 24;
    case "hour12":
    case "zero-hour12": fields.hour = numeric; return numeric >= 0 && numeric <= 12;
    case "minute":
    case "zero-minute": fields.minute = numeric; return numeric < 60;
    case "second":
    case "zero-second": return consumeSecond(fields, source);
    case "upper-meridiem":
      fields.morning = source === "AM";
      fields.evening = source === "PM";
      return fields.morning || fields.evening;
    case "lower-meridiem":
      fields.morning = source === "am";
      fields.evening = source === "pm";
      return fields.morning || fields.evening;
    case "fraction-zero":
    case "fraction-nine": fields.nanosecond = fractionNanoseconds(source); return true;
    case "zone": return consumeNumericZone(fields, source, false);
    case "iso-zone": return consumeNumericZone(fields, source, true);
    case "zone-name": return consumeZoneName(fields, source);
    case "literal": return true;
  }
}

function consumeSecond(fields: ParsedFields, source: string): boolean {
  const separator = source.search(/[.,]/u);
  const seconds = Number(separator < 0 ? source : source.slice(0, separator));
  if (seconds < 0 || seconds >= 60) {
    return false;
  }
  fields.second = seconds;
  if (separator >= 0) {
    fields.nanosecond = fractionNanoseconds(source.slice(separator));
  }
  return true;
}

function consumeNumericZone(
  fields: ParsedFields,
  source: string,
  permitsZulu: boolean,
): boolean {
  if (permitsZulu && source === "Z") {
    fields.offsetSeconds = 0;
    fields.zoneName = "UTC";
    return true;
  }
  const sign = source[0];
  if (sign !== "+" && sign !== "-") {
    return false;
  }
  const digits = source.slice(1).replaceAll(":", "");
  if (digits.length !== 2 && digits.length !== 4 && digits.length !== 6) {
    return false;
  }
  const hours = Number(digits.slice(0, 2));
  const minutes = digits.length >= 4 ? Number(digits.slice(2, 4)) : 0;
  const seconds = digits.length === 6 ? Number(digits.slice(4, 6)) : 0;
  if (hours > 24 || minutes > 60 || seconds > 60) {
    return false;
  }
  fields.offsetSeconds = (hours * 3_600 + minutes * 60 + seconds)
    * (sign === "-" ? -1 : 1);
  fields.zoneName = undefined;
  return true;
}

function consumeZoneName(fields: ParsedFields, source: string): boolean {
  fields.zoneName = source;
  if (source === "UTC") {
    fields.offsetSeconds = 0;
    return true;
  }
  const gmt = /^GMT([+-])(\d{1,2})$/u.exec(source);
  const sign = gmt?.[1];
  const hours = gmt?.[2];
  if ((sign === "+" || sign === "-") && hours !== undefined) {
    fields.offsetSeconds = Number(hours) * 3_600 * (sign === "-" ? -1 : 1);
  } else {
    fields.offsetSeconds = 0;
  }
  return true;
}

function setMonth(fields: ParsedFields, month: number): boolean {
  if (month < 1 || month > 12) {
    return false;
  }
  fields.month = month;
  fields.monthSet = true;
  return true;
}

function applyYearDay(fields: ParsedFields): boolean {
  const yearDay = fields.yearDay;
  if (yearDay === undefined) {
    return true;
  }
  const maximum = isLeapYear(fields.year) ? 366 : 365;
  if (yearDay < 1 || yearDay > maximum) {
    return false;
  }
  let month = 1;
  let day = yearDay;
  while (day > daysInMonth(fields.year, month)) {
    day -= daysInMonth(fields.year, month);
    month += 1;
  }
  if ((fields.monthSet && fields.month !== month)
    || (fields.daySet && fields.day !== day)) {
    return false;
  }
  fields.month = month;
  fields.day = day;
  return true;
}

function tokenizeLayout(layout: string): LayoutPart[] {
  const parts: LayoutPart[] = [];
  let literalStart = 0;
  let index = 0;
  while (index < layout.length) {
    const selected = tokenAt(layout, index);
    if (selected === undefined) {
      index += 1;
      continue;
    }
    if (literalStart < index) {
      parts.push({ token: "literal", text: layout.slice(literalStart, index), width: 0 });
    }
    parts.push(selected);
    index += selected.text.length;
    literalStart = index;
  }
  if (literalStart < layout.length) {
    parts.push({ token: "literal", text: layout.slice(literalStart), width: 0 });
  }
  return parts;
}

function tokenAt(layout: string, index: number): LayoutPart | undefined {
  const source = layout.slice(index);
  const fixed: ReadonlyArray<readonly [string, TokenKind]> = [
    ["January", "long-month"], ["Monday", "long-weekday"],
    ["Z07:00:00", "iso-zone"], ["-07:00:00", "zone"],
    ["Z070000", "iso-zone"], ["-070000", "zone"],
    ["Z07:00", "iso-zone"], ["-07:00", "zone"],
    ["Z0700", "iso-zone"], ["-0700", "zone"],
    ["2006", "year"], ["__2", "space-year-day"], ["002", "zero-year-day"],
    ["Z07", "iso-zone"], ["-07", "zone"],
    ["MST", "zone-name"], ["Jan", "month"], ["Mon", "weekday"],
    ["15", "hour"], ["03", "zero-hour12"], ["04", "zero-minute"],
    ["05", "zero-second"], ["06", "short-year"], ["01", "zero-month"],
    ["02", "zero-day"], ["_2", "space-day"], ["PM", "upper-meridiem"],
    ["pm", "lower-meridiem"], ["1", "numeric-month"], ["2", "day"],
    ["3", "hour12"], ["4", "minute"], ["5", "second"],
  ];
  for (const [text, token] of fixed) {
    if (!source.startsWith(text)) {
      continue;
    }
    if ((text === "Jan" || text === "Mon")
      && startsWithLowercase(source.slice(text.length))) {
      continue;
    }
    if (text === "_2" && source.startsWith("_2006")) {
      continue;
    }
    return { token, text, width: 0 };
  }
  const separator = source[0];
  const digit = source[1];
  if ((separator === "." || separator === ",")
    && (digit === "0" || digit === "9")) {
    let end = 1;
    while (source[end] === digit) {
      end += 1;
    }
    if (!isDigit(source[end])) {
      return {
        token: digit === "0" ? "fraction-zero" : "fraction-nine",
        text: source.slice(0, end),
        width: end - 1,
      };
    }
  }
  return undefined;
}

function patternFor(part: LayoutPart, next: LayoutPart | undefined): string {
  switch (part.token) {
    case "literal": return literalPattern(part.text);
    case "long-month": return "([A-Za-z]{3,9})";
    case "month": return "([A-Za-z]{3})";
    case "long-weekday": return "([A-Za-z]{6,9})";
    case "weekday": return "([A-Za-z]{3})";
    case "year": return "([0-9]{4})";
    case "short-year": return "([0-9]{2})";
    case "numeric-month":
    case "day":
    case "hour":
    case "hour12":
    case "minute": return "([0-9]{1,2})";
    case "zero-month":
    case "zero-day":
    case "zero-hour12":
    case "zero-minute": return "([0-9]{2})";
    case "space-day": return "( ?[0-9]{1,2})";
    case "space-year-day": return "( {0,2}[0-9]{1,3})";
    case "zero-year-day": return "([0-9]{3})";
    case "second":
    case "zero-second": {
      const width = part.token === "zero-second" ? "2" : "1,2";
      const fraction = next?.token === "fraction-zero" || next?.token === "fraction-nine"
        ? ""
        : "(?:[.,][0-9]+)?";
      return `([0-9]{${width}}${fraction})`;
    }
    case "upper-meridiem": return "(AM|PM)";
    case "lower-meridiem": return "(am|pm)";
    case "zone-name": return "(UTC|[A-Z]{3,5}|ChST|MeST|GMT[+-][0-9]{1,2})";
    case "zone": return `(${zonePattern(part.text, false)})`;
    case "iso-zone": return `(${zonePattern(part.text, true)})`;
    case "fraction-zero": return `([.,][0-9]{${part.width}})`;
    case "fraction-nine": return "([.,][0-9]+)?";
  }
}

function zonePattern(layout: string, zulu: boolean): string {
  const numeric = layout.startsWith("Z") ? layout.slice(1) : layout.slice(1);
  const body = numeric
    .replaceAll("07", "[0-9]{2}")
    .replaceAll("00", "[0-9]{2}");
  return `${zulu ? "Z|" : ""}[+-]${body}`;
}

function literalPattern(literal: string): string {
  let result = "";
  for (let index = 0; index < literal.length; ) {
    if (literal[index] === " ") {
      while (literal[index] === " ") {
        index += 1;
      }
      result += " +";
      continue;
    }
    result += escapeRegularExpression(literal[index] ?? "");
    index += 1;
  }
  return result;
}

function escapeRegularExpression(value: string): string {
  return /[\\^$.*+?()[\]{}|]/u.test(value) ? `\\${value}` : value;
}

function fractionNanoseconds(source: string): number {
  const digits = source.slice(1);
  return Number((digits + "000000000").slice(0, 9));
}

function startsWithLowercase(source: string): boolean {
  const first = source[0];
  return first !== undefined && first >= "a" && first <= "z";
}

function nameIndex(names: readonly string[], source: string): number {
  const selected = source.toLowerCase();
  for (let index = 0; index < names.length; index += 1) {
    if (names[index] === selected) {
      return index;
    }
  }
  return -1;
}

function isDigit(value: string | undefined): boolean {
  return value !== undefined && value >= "0" && value <= "9";
}

function rejected(
  layout: string,
  source: string,
  layoutElement: string,
  valueElement: string,
): [Time, GoError] {
  return [
    new Time(),
    new ParseError(layout, source, layoutElement, valueElement, ""),
  ];
}

function daysInMonth(year: number, month: number): number {
  switch (month) {
    case 2: return isLeapYear(year) ? 29 : 28;
    case 4:
    case 6:
    case 9:
    case 11: return 30;
    default: return 31;
  }
}

function isLeapYear(year: number): boolean {
  return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
}
