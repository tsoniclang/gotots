import { GoMapHash } from "@gotots/runtime/map.js";
import type { gostring, int64 } from "@gotots/gostdlib/internal/scalars.js";

import {
  Duration,
  ParseError,
  Ticker,
  Time,
  Timer,
} from "../../time.js";
import {
  timeRepresentationAssign,
  timeRepresentationCopy,
  timeRepresentationEqual,
  timeRepresentationHash,
} from "../portable/time/time.js";

export class TimeDurationValueOperations {
  static $project(source: Duration): int64 {
    return source.Nanoseconds();
  }

  static $wrap(source: int64): Duration {
    return new Duration(source);
  }
}

export type TimeStorage = Time;

export class TimeOperations {
  static $zero(): Time {
    return new Time();
  }

  static $copy(source: Time): Time {
    return timeRepresentationCopy(source);
  }

  static $assign(target: Time, source: Time): void {
    timeRepresentationAssign(target, source);
  }

  static $equal(left: Time, right: Time): boolean {
    return timeRepresentationEqual(left, right);
  }

  static $hash(source: Time): number {
    return timeRepresentationHash(source);
  }

  static $storageOf(source: Time): TimeStorage {
    return source;
  }

  static $fromStorage(source: TimeStorage): Time {
    return source;
  }
}

export type TimeParseErrorStorage = ParseError;

export class TimeParseErrorOperations {
  static $make(
    layout: gostring,
    value: gostring,
    layoutElement: gostring,
    valueElement: gostring,
    message: gostring,
  ): ParseError {
    return new ParseError(
      layout,
      value,
      layoutElement,
      valueElement,
      message,
    );
  }

  static $hash(source: ParseError): number {
    let hash = 2166136261;
    hash = GoMapHash.mix(hash, GoMapHash.string(source.Layout));
    hash = GoMapHash.mix(hash, GoMapHash.string(source.Value));
    hash = GoMapHash.mix(hash, GoMapHash.string(source.LayoutElem));
    hash = GoMapHash.mix(hash, GoMapHash.string(source.ValueElem));
    hash = GoMapHash.mix(hash, GoMapHash.string(source.Message));
    return hash;
  }

  static $equal(left: ParseError, right: ParseError): boolean {
    return (
      left.Layout === right.Layout &&
      left.Value === right.Value &&
      left.LayoutElem === right.LayoutElem &&
      left.ValueElem === right.ValueElem &&
      left.Message === right.Message
    );
  }

  static $storageOf(source: ParseError): TimeParseErrorStorage {
    return source;
  }

  static $fromStorage(source: TimeParseErrorStorage): ParseError {
    return source;
  }
}

export type TimeTickerStorage = Ticker;

export class TimeTickerOperations {
  static $storageOf(source: Ticker): TimeTickerStorage {
    return source;
  }

  static $fromStorage(source: TimeTickerStorage): Ticker {
    return source;
  }
}

export type TimeTimerStorage = Timer;

export class TimeTimerOperations {
  static $storageOf(source: Timer): TimeTimerStorage {
    return source;
  }

  static $fromStorage(source: TimeTimerStorage): Timer {
    return source;
  }
}
