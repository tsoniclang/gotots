import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { GoPanic } from "@gotots/runtime/panic.js";

type ReceiveResult<T> = [T, boolean];

interface ReceiveSelectCase {
  ready(): boolean;
  commit(): boolean | object;
}

export class ProviderChannel<T> implements GoReceiveChannel<T> {
  readonly #values: Array<{ readonly value: T }> = [];
  readonly #closeObservers = new Set<() => void>();
  #closed = false;

  constructor(
    private readonly zero: () => T,
    private readonly copy: (value: T) => T,
    private readonly capacity: number,
  ) {}

  $length(): number {
    return this.#values.length;
  }

  $capacity(): number {
    return this.capacity;
  }

  receive(): ReceiveResult<T> {
    const immediate = this.#take();
    if (immediate === undefined) {
      GoPanic.raiseRuntime("serial channel receive would block");
    }
    return immediate;
  }

  offer(value: T): boolean {
    if (this.#closed) {
      return false;
    }
    const prepared = this.copy(value);
    if (this.#values.length >= this.capacity) {
      return false;
    }
    this.#values.push({ value: prepared });
    return true;
  }

  close(): void {
    if (this.#closed) {
      return;
    }
    this.#closed = true;
    for (const observer of this.#closeObservers) {
      observer();
    }
    this.#closeObservers.clear();
  }

  discard(): void {
    this.#values.splice(0);
  }

  get closed(): boolean {
    return this.#closed;
  }

  $observeClose(observer: () => void): () => void {
    if (this.#closed) {
      observer();
      return (): void => undefined;
    }
    this.#closeObservers.add(observer);
    return (): void => {
      this.#closeObservers.delete(observer);
    };
  }

  $selectReceive(accept: (value: T, ok: boolean) => void): ReceiveSelectCase {
    return {
      ready: (): boolean => this.#values.length !== 0 || this.#closed,
      commit: (): boolean => {
        const result = this.#take();
        if (result === undefined) {
          return false;
        }
        accept(result[0], result[1]);
        return true;
      },
    };
  }

  #take(): ReceiveResult<T> | undefined {
    const entry = this.#values.shift();
    if (entry !== undefined) {
      return [entry.value, true];
    }
    return this.#closed ? [this.zero(), false] : undefined;
  }
}
