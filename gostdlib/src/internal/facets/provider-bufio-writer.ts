import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import { byteSlice } from "../runtime/slice.js";
import type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

const defaultBufferSize = 4096;

class WriterBuffer<Failure> {
  readonly #values: number[] = [];
  readonly #shortWrite: Failure;
  #pendingFailure: Failure | undefined;

  constructor(shortWrite: Failure) {
    this.#shortWrite = shortWrite;
  }

  get length(): number {
    return this.#values.length;
  }

  get available(): number {
    return defaultBufferSize - this.#values.length;
  }

  get failure(): Failure | undefined {
    return this.#pendingFailure;
  }

  source(): RuntimeSlice<uint8> {
    return byteSlice(this.#values);
  }

  acceptFailure(failure: Failure | undefined): void {
    this.#pendingFailure = failure;
  }

  finishFlush(
    count: int64,
    reportedFailure: Failure | undefined,
  ): Failure | undefined {
    const bufferedLength = this.#values.length;
    const failure = count < bufferedLength && reportedFailure === undefined
      ? this.#shortWrite
      : reportedFailure;
    if (failure !== undefined) {
      if (count > 0) {
        this.#values.splice(0, Math.min(count, bufferedLength));
      }
      this.#pendingFailure = failure;
      return failure;
    }
    this.#values.splice(0, bufferedLength);
    return undefined;
  }

  append(source: RuntimeSlice<uint8>, offset: number, count: number): void {
    for (let index = 0; index < count; index += 1) {
      this.#values.push(source.get(offset + index));
    }
  }

  appendByte(value: uint8): void {
    this.#values.push(value);
  }
}

export class CanonicalWriterSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
> {
  readonly #state: WriterBuffer<Failure>;
  readonly #target: Target | undefined;

  constructor(
    target: Target | undefined,
    shortWrite: Failure,
  ) {
    this.#target = target;
    this.#state = new WriterBuffer(shortWrite);
  }

  static Flush<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriterTargetSync<Failure>,
  >(
    receiver: CanonicalWriterSync<Failure, Target> | undefined,
    recovery?: GoRecovery,
  ): Failure | undefined {
    return requireWriterSync(receiver).Flush(recovery);
  }

  static Write<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriterTargetSync<Failure>,
  >(
    receiver: CanonicalWriterSync<Failure, Target> | undefined,
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined] {
    return requireWriterSync(receiver).Write(source, recovery);
  }

  static WriteByte<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriterTargetSync<Failure>,
  >(
    receiver: CanonicalWriterSync<Failure, Target> | undefined,
    value: uint8,
    recovery?: GoRecovery,
  ): Failure | undefined {
    return requireWriterSync(receiver).WriteByte(value, recovery);
  }

  Flush(recovery?: GoRecovery): Failure | undefined {
    if (this.#state.failure !== undefined || this.#state.length === 0) {
      return this.#state.failure;
    }
    const [count, failure] = requireTarget(this.#target).Write(
      this.#state.source(),
      recovery,
    );
    return this.#state.finishFlush(count, failure);
  }

  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined] {
    if (this.#state.failure !== undefined) {
      return [0, this.#state.failure];
    }
    let accepted = 0;
    while (source.length - accepted > this.#state.available &&
      this.#state.failure === undefined) {
      let count: int64;
      if (this.#state.length === 0) {
        const result = requireTarget(this.#target).Write(
          source.slice(accepted, source.length, null),
          recovery,
        );
        count = result[0];
        this.#state.acceptFailure(result[1]);
      } else {
        count = Math.min(this.#state.available, source.length - accepted);
        this.#state.append(source, accepted, count);
        this.Flush(recovery);
      }
      accepted += count;
    }
    if (this.#state.failure !== undefined) {
      return [accepted, this.#state.failure];
    }
    const count = source.length - accepted;
    this.#state.append(source, accepted, count);
    return [accepted + count, undefined];
  }

  WriteByte(value: uint8, recovery?: GoRecovery): Failure | undefined {
    if (this.#state.failure !== undefined) {
      return this.#state.failure;
    }
    if (this.#state.available === 0) {
      const failure = this.Flush(recovery);
      if (failure !== undefined) {
        return failure;
      }
    }
    this.#state.appendByte(value);
    return undefined;
  }
}

export function NewWriterCanonicalSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  target: Target | undefined,
  shortWrite: Failure | undefined,
): CanonicalWriterSync<Failure, Target> {
  return new CanonicalWriterSync(target, requireShortWrite(shortWrite));
}

export class CanonicalWriterAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
> {
  readonly #state: WriterBuffer<Failure>;
  readonly #target: Target | undefined;

  constructor(
    target: Target | undefined,
    shortWrite: Failure,
  ) {
    this.#target = target;
    this.#state = new WriterBuffer(shortWrite);
  }

  static Flush<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriterTargetAsync<Failure>,
  >(
    receiver: CanonicalWriterAsync<Failure, Target> | undefined,
    recovery?: GoRecovery,
  ): Promise<Failure | undefined> {
    return requireWriterAsync(receiver).Flush(recovery);
  }

  static Write<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriterTargetAsync<Failure>,
  >(
    receiver: CanonicalWriterAsync<Failure, Target> | undefined,
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]> {
    return requireWriterAsync(receiver).Write(source, recovery);
  }

  static WriteByte<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriterTargetAsync<Failure>,
  >(
    receiver: CanonicalWriterAsync<Failure, Target> | undefined,
    value: uint8,
    recovery?: GoRecovery,
  ): Promise<Failure | undefined> {
    return requireWriterAsync(receiver).WriteByte(value, recovery);
  }

  async Flush(recovery?: GoRecovery): Promise<Failure | undefined> {
    if (this.#state.failure !== undefined || this.#state.length === 0) {
      return this.#state.failure;
    }
    const [count, failure] = await requireTarget(this.#target).Write(
      this.#state.source(),
      recovery,
    );
    return this.#state.finishFlush(count, failure);
  }

  async Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]> {
    if (this.#state.failure !== undefined) {
      return [0, this.#state.failure];
    }
    let accepted = 0;
    while (source.length - accepted > this.#state.available &&
      this.#state.failure === undefined) {
      let count: int64;
      if (this.#state.length === 0) {
        const result = await requireTarget(this.#target).Write(
          source.slice(accepted, source.length, null),
          recovery,
        );
        count = result[0];
        this.#state.acceptFailure(result[1]);
      } else {
        count = Math.min(this.#state.available, source.length - accepted);
        this.#state.append(source, accepted, count);
        await this.Flush(recovery);
      }
      accepted += count;
    }
    if (this.#state.failure !== undefined) {
      return [accepted, this.#state.failure];
    }
    const count = source.length - accepted;
    this.#state.append(source, accepted, count);
    return [accepted + count, undefined];
  }

  async WriteByte(
    value: uint8,
    recovery?: GoRecovery,
  ): Promise<Failure | undefined> {
    if (this.#state.failure !== undefined) {
      return this.#state.failure;
    }
    if (this.#state.available === 0) {
      const failure = await this.Flush(recovery);
      if (failure !== undefined) {
        return failure;
      }
    }
    this.#state.appendByte(value);
    return undefined;
  }
}

export function NewWriterCanonicalAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  target: Target | undefined,
  shortWrite: Failure | undefined,
): CanonicalWriterAsync<Failure, Target> {
  return new CanonicalWriterAsync(target, requireShortWrite(shortWrite));
}

function requireTarget<Target>(target: Target | undefined): Target {
  if (target === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return target;
}

function requireShortWrite<Failure>(failure: Failure | undefined): Failure {
  if (failure === undefined) {
    GoPanic.raiseRuntime("gostdlib provider supplied a nil io.ErrShortWrite");
  }
  return failure;
}

function requireWriterSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  receiver: CanonicalWriterSync<Failure, Target> | undefined,
): CanonicalWriterSync<Failure, Target> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

function requireWriterAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  receiver: CanonicalWriterAsync<Failure, Target> | undefined,
): CanonicalWriterAsync<Failure, Target> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}
