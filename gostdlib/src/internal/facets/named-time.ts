import { Ticker, Time, Timer } from "../../time.js";
import {
  timeRepresentationEqual,
  timeRepresentationHash,
} from "../portable/time/time.js";

export type TimeStorage = Time;

export class TimeOperations {
  static $zero(): Time {
    return new Time();
  }

  static $copy(source: Time): Time {
    return source;
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
