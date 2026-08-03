import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable } from "@gotots/runtime/scalars.js";

export class Once {
  #state: "idle" | "running" | "done" = "idle";
  #completion: Promise<void> = Promise.resolve();

  static async Do(
    receiver: Once | undefined,
    f: (() => Awaitable<void>) | undefined,
  ): Promise<void> {
    if (receiver === undefined) {
      throw new TypeError("Once.Do called with nil receiver");
    }
    if (receiver.#state === "done") {
      return;
    }
    if (receiver.#state === "running") {
      await receiver.#completion.catch(() => undefined);
      return;
    }
    receiver.#state = "running";
    receiver.#completion = (async (): Promise<void> => {
      try {
        if (f === undefined) {
          GoPanic.raiseRuntime("sync.Once.Do called with nil function");
        }
        await f();
      } finally {
        receiver.#state = "done";
      }
    })();
    await receiver.#completion;
  }
}

export function OnceFunc(
  f: (() => Awaitable<void>) | undefined,
): () => Promise<void> {
  let result: Promise<void> | undefined;
  return (): Promise<void> => {
    result ??= f === undefined
      ? Promise.reject(GoPanic.createRuntime("sync.OnceFunc called with nil function"))
      : Promise.resolve(f());
    return result;
  };
}

export function OnceValue<T>(
  f: (() => Awaitable<T>) | undefined,
): () => Promise<T> {
  let result: Promise<T> | undefined;
  return (): Promise<T> => {
    result ??= f === undefined
      ? Promise.reject(GoPanic.createRuntime("sync.OnceValue called with nil function"))
      : Promise.resolve().then(f);
    return result;
  };
}
