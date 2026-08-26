import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export class Once {
  #state: "idle" | "running" | "done" = "idle";

  static $copy(source: Once): Once {
    const result = new Once();
    result.#state = source.#state;
    return result;
  }

  static $equal(left: Once, right: Once): boolean {
    return left.#state === right.#state;
  }

  static $hash(source: Once): number {
    return GoMapHash.string(source.#state);
  }

  static Do(
    receiver: Once | undefined,
    f: (() => void) | undefined,
  ): void {
    if (receiver === undefined) {
      throw new TypeError("Once.Do called with nil receiver");
    }
    if (receiver.#state === "done") {
      return;
    }
    if (receiver.#state === "running") {
      GoPanic.raiseRuntime(
        "sync: Once.Do would block under serial execution",
      );
    }
    receiver.#state = "running";
    try {
      if (f === undefined) {
        GoPanic.raiseRuntime("sync.Once.Do called with nil function");
      }
      f();
    } finally {
      receiver.#state = "done";
    }
  }
}

export function OnceFunc(
  f: (() => void) | undefined,
): () => void {
  let state: "idle" | "running" | "done" | "panicked" = "idle";
  let panicValue: unknown;
  return (): void => {
    if (state === "panicked") {
      throw panicValue;
    }
    if (state === "done") {
      return;
    }
    if (state === "running") {
      GoPanic.raiseRuntime(
        "sync: OnceFunc would block under serial execution",
      );
    }
    state = "running";
    try {
      if (f === undefined) {
        GoPanic.raiseRuntime("sync.OnceFunc called with nil function");
      }
      f();
      state = "done";
    } catch (failure) {
      panicValue = failure;
      state = "panicked";
      throw failure;
    }
  };
}

export function OnceValue<T>(
  f: (() => T) | undefined,
): () => T {
  type Outcome =
    | { readonly kind: "idle" }
    | { readonly kind: "running" }
    | { readonly kind: "done"; readonly value: T }
    | { readonly kind: "panicked"; readonly value: unknown };
  let outcome: Outcome = { kind: "idle" };
  return (): T => {
    if (outcome.kind === "panicked") {
      throw outcome.value;
    }
    if (outcome.kind === "done") {
      return outcome.value;
    }
    if (outcome.kind === "running") {
      GoPanic.raiseRuntime(
        "sync: OnceValue would block under serial execution",
      );
    }
    outcome = { kind: "running" };
    try {
      if (f === undefined) {
        GoPanic.raiseRuntime("sync.OnceValue called with nil function");
      }
      const value = f();
      outcome = { kind: "done", value };
      return value;
    } catch (failure) {
      outcome = { kind: "panicked", value: failure };
      throw failure;
    }
  };
}
