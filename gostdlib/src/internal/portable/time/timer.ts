import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool } from "@gotots/gostdlib/internal/scalars.js";
import { hostInteger } from "../../host-integer.js";
import {
  cancelSchedule,
  schedule,
  type ClockHandle,
} from "../../node/time/clock.js";
import { ProviderChannel } from "../concurrency/channel.js";
import { Duration } from "./duration.js";
import { Now, Time } from "./time.js";

function delay(d: Duration): number {
  return hostInteger(d.Nanoseconds()) / 1_000_000;
}

let assignTimerRepresentation: (target: Timer, source: Timer) => void;
let copyTimerRepresentation: (source: Timer) => Timer;
let equalTimerRepresentation: (left: Timer, right: Timer) => bool;
let hashTimerRepresentation: (source: Timer) => number;
let assignTickerRepresentation: (target: Ticker, source: Ticker) => void;
let copyTickerRepresentation: (source: Ticker) => Ticker;

export class Timer {
  C: GoReceiveChannel<Time> | undefined;
  readonly #channel: ProviderChannel<Time> | undefined;
  #handle: ClockHandle | undefined;
  #active = false;
  #initialized = false;
  readonly #action: (() => Awaitable<void>) | undefined;

  constructor(duration?: Duration, action?: () => Awaitable<void>) {
    this.#action = action;
    if (duration !== undefined && action === undefined) {
      this.#channel = new ProviderChannel<Time>(
        () => new Time(),
        (value) => value,
        1,
      );
      this.C = this.#channel;
    }
    if (duration !== undefined) {
      this.#initialized = true;
      this.#start(duration);
    }
  }

  static {
    assignTimerRepresentation = (target: Timer, source: Timer): void => {
      target.C = source.C;
      target.#initialized = source.#initialized;
    };
    copyTimerRepresentation = (source: Timer): Timer => {
      const result = new Timer();
      assignTimerRepresentation(result, source);
      return result;
    };
    equalTimerRepresentation = (left: Timer, right: Timer): bool =>
      left.C === right.C && left.#initialized === right.#initialized;
    hashTimerRepresentation = (source: Timer): number => {
      const channelHash = source.C === undefined
        ? 0
        : GoMapHash.object(source.C);
      return GoMapHash.mix(
        channelHash,
        GoMapHash.boolean(source.#initialized),
      );
    };
  }

  static Stop(receiver: Timer | undefined): bool {
    const timer = requireTimer(receiver);
    if (!timer.#initialized) {
      GoPanic.raiseRuntime("time: Stop called on uninitialized Timer");
    }
    return timer.#stop();
  }

  static Reset(receiver: Timer | undefined, d: Duration): bool {
    const timer = requireTimer(receiver);
    if (!timer.#initialized) {
      GoPanic.raiseRuntime("time: Reset called on uninitialized Timer");
    }
    const active = timer.#stop();
    timer.#channel?.discard();
    timer.#start(d);
    return active;
  }

  #start(duration: Duration): void {
    this.#active = true;
    this.#handle = schedule(delay(duration), () => {
      this.#active = false;
      this.#handle = undefined;
      if (this.#action === undefined) {
        this.#channel?.offer(Now());
      } else {
        void this.#action();
      }
    });
  }

  #stop(): boolean {
    if (!this.#active || this.#handle === undefined) {
      return false;
    }
    cancelSchedule(this.#handle);
    this.#handle = undefined;
    this.#active = false;
    return true;
  }
}

export function timerRepresentationAssign(target: Timer, source: Timer): void {
  assignTimerRepresentation(target, source);
}

export function timerRepresentationCopy(source: Timer): Timer {
  return copyTimerRepresentation(source);
}

export function timerRepresentationEqual(left: Timer, right: Timer): bool {
  return equalTimerRepresentation(left, right);
}

export function timerRepresentationHash(source: Timer): number {
  return hashTimerRepresentation(source);
}

export class Ticker {
  C: GoReceiveChannel<Time>;
  #runtime: TickerRuntime | undefined;
  #initialized = false;

  constructor(duration?: Duration) {
    const channel = new ProviderChannel<Time>(
      () => new Time(),
      (value) => value,
      1,
    );
    this.C = channel;
    if (duration === undefined) {
      return;
    }
    if (duration.Nanoseconds() <= 0n) {
      GoPanic.raiseRuntime("non-positive interval for NewTicker");
    }
    this.#initialized = true;
    this.#runtime = {
      channel,
      duration,
      handle: undefined,
      stopped: false,
    };
    scheduleTicker(this.#runtime);
  }

  static {
    assignTickerRepresentation = (target: Ticker, source: Ticker): void => {
      target.C = source.C;
      target.#runtime = source.#runtime;
      target.#initialized = source.#initialized;
    };
    copyTickerRepresentation = (source: Ticker): Ticker => {
      const result = new Ticker();
      assignTickerRepresentation(result, source);
      return result;
    };
  }

  static Stop(receiver: Ticker | undefined): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("Ticker.Stop called with nil receiver");
    }
    if (!receiver.#initialized || receiver.#runtime === undefined) {
      return;
    }
    receiver.#runtime.stopped = true;
    if (receiver.#runtime.handle !== undefined) {
      cancelSchedule(receiver.#runtime.handle);
      receiver.#runtime.handle = undefined;
    }
  }
}

interface TickerRuntime {
  readonly channel: ProviderChannel<Time>;
  readonly duration: Duration;
  handle: ClockHandle | undefined;
  stopped: boolean;
}

function scheduleTicker(runtime: TickerRuntime): void {
  runtime.handle = schedule(delay(runtime.duration), () => {
    runtime.handle = undefined;
    if (!runtime.stopped) {
      runtime.channel.offer(Now());
      scheduleTicker(runtime);
    }
  });
}

export function tickerRepresentationAssign(target: Ticker, source: Ticker): void {
  assignTickerRepresentation(target, source);
}

export function tickerRepresentationCopy(source: Ticker): Ticker {
  return copyTickerRepresentation(source);
}

export function After(d: Duration): GoReceiveChannel<Time> {
  const channel = new Timer(d).C;
  if (channel === undefined) {
    return GoPanic.raiseRuntime("time.After created a timer without a channel");
  }
  return channel;
}

export function AfterFunc(
  d: Duration,
  f: (() => Awaitable<void>) | undefined,
): Timer {
  const invoke = f ?? (() => {
    GoPanic.raiseRuntime("time.AfterFunc called with nil function");
  });
  return new Timer(d, invoke);
}

export function NewTicker(d: Duration): Ticker {
  return new Ticker(d);
}

export function NewTimer(d: Duration): Timer {
  return new Timer(d);
}

export function Sleep(d: Duration): Promise<void> {
  if (d.Nanoseconds() <= 0n) {
    return Promise.resolve();
  }
  return new Promise<void>((resolve) => {
    schedule(delay(d), resolve);
  });
}

function requireTimer(receiver: Timer | undefined): Timer {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("Timer method called with nil receiver");
  }
  return receiver;
}
