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
  tickerRepresentationAssign,
  tickerRepresentationCopy,
  timerRepresentationAssign,
  timerRepresentationCopy,
  timerRepresentationEqual,
  timerRepresentationHash,
} from "../portable/time/timer.js";
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

  static $copy(source: ParseError): ParseError {
    return new ParseError(
      source.Layout,
      source.Value,
      source.LayoutElem,
      source.ValueElem,
      source.Message,
    );
  }

  static $assign(target: ParseError, source: ParseError): void {
    target.Layout = source.Layout;
    target.Value = source.Value;
    target.LayoutElem = source.LayoutElem;
    target.ValueElem = source.ValueElem;
    target.Message = source.Message;
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
  static $assign(target: Ticker, source: Ticker): void {
    tickerRepresentationAssign(target, source);
  }

  static $copy(source: Ticker): Ticker {
    return tickerRepresentationCopy(source);
  }

  static $storageOf(source: Ticker): TimeTickerStorage {
    return source;
  }

  static $fromStorage(source: TimeTickerStorage): Ticker {
    return source;
  }
}

export type TimeTimerStorage = Timer;

export class TimeTimerOperations {
  static $assign(target: Timer, source: Timer): void {
    timerRepresentationAssign(target, source);
  }

  static $copy(source: Timer): Timer {
    return timerRepresentationCopy(source);
  }

  static $equal(left: Timer, right: Timer): boolean {
    return timerRepresentationEqual(left, right);
  }

  static $hash(source: Timer): number {
    return timerRepresentationHash(source);
  }

  static $storageOf(source: Timer): TimeTimerStorage {
    return source;
  }

  static $fromStorage(source: TimeTimerStorage): Timer {
    return source;
  }
}
