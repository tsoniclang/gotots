import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/runtime/scalars.js";
import {
  cancelSchedule,
  schedule,
  type ClockHandle,
} from "../../node/time/clock.js";
import { ProviderChannel } from "../concurrency/channel.js";
import { Duration } from "./duration.js";
import { Now, Time } from "./time.js";

function delay(d: Duration): number {
  return d.Nanoseconds() / 1_000_000;
}

export class Timer {
  readonly C: GoReceiveChannel<Time> | undefined;
  readonly #channel: ProviderChannel<Time> | undefined;
  #handle: ClockHandle | undefined;
  #active = false;
  readonly #action: (() => Promise<void>) | undefined;

  constructor(duration?: Duration, action?: () => Promise<void>) {
    this.#action = action;
    if (action === undefined) {
      this.#channel = new ProviderChannel<Time>(
        () => new Time(),
        (value) => value,
        1,
      );
      this.C = this.#channel;
    }
    if (duration !== undefined) {
      this.#start(duration);
    }
  }

  static Stop(receiver: Timer | undefined): bool {
    return requireTimer(receiver).#stop();
  }

  static Reset(receiver: Timer | undefined, d: Duration): bool {
    const timer = requireTimer(receiver);
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

export class Ticker {
  readonly #channel = new ProviderChannel<Time>(
    () => new Time(),
    (value) => value,
    1,
  );
  readonly C: GoReceiveChannel<Time> = this.#channel;
  #handle: ClockHandle | undefined;
  #stopped = false;

  constructor(private readonly duration: Duration) {
    if (duration.Nanoseconds() <= 0) {
      GoPanic.raiseRuntime("non-positive interval for NewTicker");
    }
    this.#schedule();
  }

  static Stop(receiver: Ticker | undefined): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("Ticker.Stop called with nil receiver");
    }
    receiver.#stopped = true;
    if (receiver.#handle !== undefined) {
      cancelSchedule(receiver.#handle);
      receiver.#handle = undefined;
    }
  }

  #schedule(): void {
    this.#handle = schedule(delay(this.duration), () => {
      this.#handle = undefined;
      if (!this.#stopped) {
        this.#channel.offer(Now());
        this.#schedule();
      }
    });
  }
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
  f: (() => Promise<void>) | undefined,
): Timer {
  const invoke = f ?? (async (): Promise<void> => {
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

function requireTimer(receiver: Timer | undefined): Timer {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("Timer method called with nil receiver");
  }
  return receiver;
}
