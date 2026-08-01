import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { uint8 } from "@gotots/runtime/scalars.js";

import { ParseError } from "./parse-error.js";

const layout = "2006-01-02T15:04:05Z07:00";
const shape = /^(\d{4})-(\d{2})-(\d{2})T(\d{1,2}):(\d{2}):(\d{2})([.,]\d+)?(Z|[+-]\d{2}:\d{2})$/u;

export type ParsedRFC3339 = {
  readonly ok: true;
  readonly epochMilliseconds: number;
  readonly nanosecondRemainder: number;
  readonly offsetSeconds: number;
  readonly zoneName: string | undefined;
};

export type RejectedRFC3339 = {
  readonly ok: false;
  readonly failure: ParseError;
};

export function parseRFC3339Bytes(
  source: RuntimeSlice<uint8>,
): ParsedRFC3339 | RejectedRFC3339 {
  let text = "";
  for (let index = 0; index < source.length; index += 1) {
    const byte = source.get(index);
    if (byte > 0x7f) {
      return rejected(text + String.fromCharCode(byte));
    }
    text += String.fromCharCode(byte);
  }
  return parseRFC3339(text);
}

function parseRFC3339(source: string): ParsedRFC3339 | RejectedRFC3339 {
  const match = shape.exec(source);
  if (match === null) {
    return rejected(source);
  }
  const yearText = match[1];
  const monthText = match[2];
  const dayText = match[3];
  const hourText = match[4];
  const minuteText = match[5];
  const secondText = match[6];
  const fractionText = match[7] ?? "";
  const zoneText = match[8];
  if (yearText === undefined
    || monthText === undefined
    || dayText === undefined
    || hourText === undefined
    || minuteText === undefined
    || secondText === undefined
    || zoneText === undefined) {
    return rejected(source);
  }

  const year = Number(yearText);
  const month = Number(monthText);
  const day = Number(dayText);
  const hour = Number(hourText);
  const minute = Number(minuteText);
  const second = Number(secondText);
  if (month < 1 || month > 12
    || day < 1 || day > daysInMonth(year, month)
    || hour > 23
    || minute > 59
    || second > 59) {
    return rejected(source);
  }

  const fractionDigits = fractionText === "" ? "" : fractionText.slice(1);
  const nanoseconds = Number((fractionDigits + "000000000").slice(0, 9));
  const milliseconds = Math.floor(nanoseconds / 1_000_000);
  const nanosecondRemainder = nanoseconds - milliseconds * 1_000_000;

  let offsetMinutes = 0;
  if (zoneText !== "Z") {
    const offsetHour = Number(zoneText.slice(1, 3));
    const offsetMinute = Number(zoneText.slice(4, 6));
    if (offsetHour > 24 || offsetMinute > 60) {
      return rejected(source);
    }
    offsetMinutes = (offsetHour * 60 + offsetMinute)
      * (zoneText[0] === "-" ? -1 : 1);
  }

  const date = new Date(0);
  date.setUTCFullYear(year, month - 1, day);
  date.setUTCHours(hour, minute, second, milliseconds);
  return {
    ok: true,
    epochMilliseconds: date.getTime() - offsetMinutes * 60_000,
    nanosecondRemainder,
    offsetSeconds: offsetMinutes * 60,
    zoneName: zoneText === "Z" ? "UTC" : undefined,
  };
}

function rejected(source: string): RejectedRFC3339 {
  return {
    ok: false,
    failure: new ParseError(layout, source, layout, source, ""),
  };
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
