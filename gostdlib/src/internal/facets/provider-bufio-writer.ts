import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
} from "../host-integer.js";

import { byteSlice } from "../runtime/slice.js";
import type {
  CanonicalWriter,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type {
  CanonicalError,
  CanonicalWriter,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

const defaultBufferSize = 4096;

class WriterBuffer<Failure> {
  readonly #values: number[];
  readonly #shortWrite: Failure;
  #length: number;
  #pendingFailure: Failure | undefined;

  constructor(
    shortWrite: Failure,
    values: number[] = [],
    length = 0,
    pendingFailure: Failure | undefined = undefined,
  ) {
    this.#shortWrite = shortWrite;
    this.#values = values;
    this.#length = length;
    this.#pendingFailure = pendingFailure;
  }

  copy(): WriterBuffer<Failure> {
    return new WriterBuffer(
      this.#shortWrite,
      this.#values,
      this.#length,
      this.#pendingFailure,
    );
  }

  get shortWrite(): Failure {
    return this.#shortWrite;
  }

  get length(): number {
    return this.#length;
  }

  get available(): number {
    return defaultBufferSize - this.#length;
  }

  get failure(): Failure | undefined {
    return this.#pendingFailure;
  }

  source(): RuntimeSlice<uint8> {
    return byteSlice(this.#values.slice(0, this.#length));
  }

  acceptFailure(failure: Failure | undefined): void {
    this.#pendingFailure = failure;
  }

  finishFlush(
    count: int,
    reportedFailure: Failure | undefined,
  ): Failure | undefined {
    const bufferedLength = this.#length;
    const hostCount = hostInteger(count);
    const failure = hostCount < bufferedLength && reportedFailure === undefined
      ? this.#shortWrite
      : reportedFailure;
    if (failure !== undefined) {
      if (hostCount > 0) {
        const accepted = Math.min(hostCount, bufferedLength);
        this.#values.copyWithin(0, accepted, bufferedLength);
        this.#length = bufferedLength - accepted;
      }
      this.#pendingFailure = failure;
      return failure;
    }
    this.#length = 0;
    return undefined;
  }

  append(source: RuntimeSlice<uint8>, offset: number, count: number): void {
    for (let index = 0; index < count; index += 1) {
      this.#values[this.#length + index] = source.get(offset + index);
    }
    this.#length += count;
  }

  appendByte(value: uint8): void {
    this.#values[this.#length] = value;
    this.#length += 1;
  }
}

export class CanonicalBufioWriter<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
> {
  #state: WriterBuffer<Failure>;
  #target: Target | undefined;

  constructor(
    target: Target | undefined,
    shortWrite: Failure,
  ) {
    this.#target = target;
    this.#state = new WriterBuffer(shortWrite);
  }

  static $copy<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    source: CanonicalBufioWriter<Failure, Target>,
  ): CanonicalBufioWriter<Failure, Target> {
    const target = new CanonicalBufioWriter(
      source.#target,
      source.#state.shortWrite,
    );
    target.#state = source.#state.copy();
    return target;
  }

  static $assign<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    target: CanonicalBufioWriter<Failure, Target>,
    source: CanonicalBufioWriter<Failure, Target>,
  ): void {
    const state = source.#state.copy();
    const providerTarget = source.#target;
    target.#state = state;
    target.#target = providerTarget;
  }

  static Flush<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    receiver: CanonicalBufioWriter<Failure, Target> | undefined,
    recovery?: GoRecovery,
  ): Promise<Failure | undefined> {
    return requireWriter(receiver).Flush(recovery);
  }

  static Write<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    receiver: CanonicalBufioWriter<Failure, Target> | undefined,
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int, Failure | undefined]> {
    return requireWriter(receiver).Write(source, recovery);
  }

  static WriteByte<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    receiver: CanonicalBufioWriter<Failure, Target> | undefined,
    value: uint8,
    recovery?: GoRecovery,
  ): Promise<Failure | undefined> {
    return requireWriter(receiver).WriteByte(value, recovery);
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
  ): Promise<[int, Failure | undefined]> {
    if (this.#state.failure !== undefined) {
      return [0n, this.#state.failure];
    }
    let accepted = 0;
    while (source.length - accepted > this.#state.available &&
      this.#state.failure === undefined) {
      let count: number;
      if (this.#state.length === 0) {
        const result = await requireTarget(this.#target).Write(
          source.slice(accepted, source.length, null),
          recovery,
        );
        count = hostInteger(result[0]);
        this.#state.acceptFailure(result[1]);
      } else {
        count = Math.min(this.#state.available, source.length - accepted);
        this.#state.append(source, accepted, count);
        await this.Flush(recovery);
      }
      accepted += count;
    }
    if (this.#state.failure !== undefined) {
      return [integerFromHost(accepted), this.#state.failure];
    }
    const count = source.length - accepted;
    this.#state.append(source, accepted, count);
    return [integerFromHost(accepted + count), undefined];
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

export class DirectBufioWriter<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
> {
  #state: WriterBuffer<Failure>;
  #target: Target | undefined;

  constructor(target: Target | undefined, shortWrite: Failure) {
    this.#target = target;
    this.#state = new WriterBuffer(shortWrite);
  }

  static $copy<
    Failure extends ProviderErrorInterface,
    Target extends ProviderWriterInterface<Failure>,
  >(
    source: DirectBufioWriter<Failure, Target>,
  ): DirectBufioWriter<Failure, Target> {
    const target = new DirectBufioWriter(
      source.#target,
      source.#state.shortWrite,
    );
    target.#state = source.#state.copy();
    return target;
  }

  static $assign<
    Failure extends ProviderErrorInterface,
    Target extends ProviderWriterInterface<Failure>,
  >(
    target: DirectBufioWriter<Failure, Target>,
    source: DirectBufioWriter<Failure, Target>,
  ): void {
    target.#state = source.#state.copy();
    target.#target = source.#target;
  }

  static Flush<
    Failure extends ProviderErrorInterface,
    Target extends ProviderWriterInterface<Failure>,
  >(
    receiver: DirectBufioWriter<Failure, Target> | undefined,
    recovery?: GoRecovery,
  ): Failure | undefined {
    return requireDirectWriter(receiver).Flush(recovery);
  }

  static Write<
    Failure extends ProviderErrorInterface,
    Target extends ProviderWriterInterface<Failure>,
  >(
    receiver: DirectBufioWriter<Failure, Target> | undefined,
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined] {
    return requireDirectWriter(receiver).Write(source, recovery);
  }

  static WriteByte<
    Failure extends ProviderErrorInterface,
    Target extends ProviderWriterInterface<Failure>,
  >(
    receiver: DirectBufioWriter<Failure, Target> | undefined,
    value: uint8,
    recovery?: GoRecovery,
  ): Failure | undefined {
    return requireDirectWriter(receiver).WriteByte(value, recovery);
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
  ): [int, Failure | undefined] {
    if (this.#state.failure !== undefined) {
      return [0n, this.#state.failure];
    }
    let accepted = 0;
    while (source.length - accepted > this.#state.available &&
      this.#state.failure === undefined) {
      let count: number;
      if (this.#state.length === 0) {
        const result = requireTarget(this.#target).Write(
          source.slice(accepted, source.length, null),
          recovery,
        );
        count = hostInteger(result[0]);
        this.#state.acceptFailure(result[1]);
      } else {
        count = Math.min(this.#state.available, source.length - accepted);
        this.#state.append(source, accepted, count);
        this.Flush(recovery);
      }
      accepted += count;
    }
    if (this.#state.failure !== undefined) {
      return [integerFromHost(accepted), this.#state.failure];
    }
    const count = source.length - accepted;
    this.#state.append(source, accepted, count);
    return [integerFromHost(accepted + count), undefined];
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

export class BufioWriterOperations {
  static $copy<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    source: CanonicalBufioWriter<Failure, Target>,
  ): CanonicalBufioWriter<Failure, Target> {
    return CanonicalBufioWriter.$copy(source);
  }

  static $assign<
    Failure extends GoInterfaceValue,
    Target extends CanonicalWriter<Failure>,
  >(
    target: CanonicalBufioWriter<Failure, Target>,
    source: CanonicalBufioWriter<Failure, Target>,
  ): void {
    CanonicalBufioWriter.$assign(target, source);
  }
}

export function NewWriterCanonical<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
>(
  target: Target | undefined,
  shortWrite: Failure | undefined,
): CanonicalBufioWriter<Failure, Target> {
  return new CanonicalBufioWriter(target, requireShortWrite(shortWrite));
}

export function NewWriterDirect<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  target: Target | undefined,
  shortWrite: Failure | undefined,
): DirectBufioWriter<Failure, Target> {
  return new DirectBufioWriter(target, requireShortWrite(shortWrite));
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

function requireWriter<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
>(
  receiver: CanonicalBufioWriter<Failure, Target> | undefined,
): CanonicalBufioWriter<Failure, Target> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

function requireDirectWriter<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  receiver: DirectBufioWriter<Failure, Target> | undefined,
): DirectBufioWriter<Failure, Target> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}
