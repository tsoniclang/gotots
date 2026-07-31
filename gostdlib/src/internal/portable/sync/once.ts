import { GoPanic } from "@gotots/runtime/panic.js";

export class Once {
  #state: "idle" | "running" | "done" = "idle";
  #completion: Promise<void> = Promise.resolve();

  static async Do(
    receiver: Once | undefined,
    f: (() => Promise<void>) | undefined,
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
  f: (() => Promise<void>) | undefined,
): () => Promise<void> {
  let result: Promise<void> | undefined;
  return (): Promise<void> => {
    result ??= f === undefined
      ? Promise.reject(GoPanic.createRuntime("sync.OnceFunc called with nil function"))
      : f();
    return result;
  };
}

export function OnceValue<T>(f: (() => T) | undefined): () => T {
  let complete = false;
  let value: T;
  let panic: GoPanic | undefined;
  return (): T => {
    if (!complete) {
      try {
        if (f === undefined) {
          GoPanic.raiseRuntime("sync.OnceValue called with nil function");
        }
        value = f();
        complete = true;
      } catch (caught) {
        if (caught instanceof GoPanic) {
          panic = caught;
          complete = true;
        }
        throw caught;
      }
    }
    if (panic !== undefined) {
      throw panic;
    }
    return value;
  };
}
