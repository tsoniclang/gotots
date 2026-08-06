import type { GoReceiveChannel } from "@gotots/runtime/channel.js";

type ReceiveResult<T> = [T, boolean];
type Receiver<T> = (result: ReceiveResult<T>) => boolean;
type Claim = (failure: object | undefined) => boolean;

interface ReceiveSelectCase {
  ready(): boolean;
  commit(): boolean | object;
  subscribe(claim: Claim): () => void;
}

export class ProviderChannel<T> implements GoReceiveChannel<T> {
  readonly #values: Array<{ readonly value: T }> = [];
  readonly #receivers: Array<Receiver<T>> = [];
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

  receive(): Promise<ReceiveResult<T>> {
    const immediate = this.#take();
    if (immediate !== undefined) {
      return Promise.resolve(immediate);
    }
    return new Promise<ReceiveResult<T>>((resolve) => {
      this.#receivers.push((result) => {
        resolve(result);
        return true;
      });
    });
  }

  offer(value: T): boolean {
    if (this.#closed) {
      return false;
    }
    const prepared = this.copy(value);
    while (this.#receivers.length !== 0) {
      const receiver = this.#receivers.shift();
      if (receiver !== undefined && receiver([prepared, true])) {
        return true;
      }
    }
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
    for (const receiver of this.#receivers.splice(0)) {
      receiver([this.zero(), false]);
    }
  }

  discard(): void {
    this.#values.splice(0);
  }

  get closed(): boolean {
    return this.#closed;
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
      subscribe: (claim: Claim): (() => void) => {
        const receive: Receiver<T> = (result): boolean => {
          if (!claim(undefined)) {
            return false;
          }
          accept(result[0], result[1]);
          return true;
        };
        this.#receivers.push(receive);
        return (): void => {
          const index = this.#receivers.indexOf(receive);
          if (index >= 0) {
            this.#receivers.splice(index, 1);
          }
        };
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
