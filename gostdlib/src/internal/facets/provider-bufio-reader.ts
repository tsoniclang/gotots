import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import { byteSlice, writeBytes } from "../runtime/slice.js";
import type { CanonicalReader } from "./provider-io-contract.js";

export type {
  CanonicalError,
  CanonicalReader,
} from "./provider-io-contract.js";

const defaultBufferSize = 4096;

class ReaderBuffer<Failure> {
  readonly #values: number[] = [];
  #pendingFailure: Failure | undefined;

  shouldFill(): boolean {
    return this.#values.length === 0 && this.#pendingFailure === undefined;
  }

  accept(
    source: RuntimeSlice<uint8>,
    count: int64,
    failure: Failure | undefined,
  ): void {
    for (let index = 0; index < count; index += 1) {
      this.#values.push(source.get(index));
    }
    this.#pendingFailure = failure;
  }

  setFailure(failure: Failure): void {
    this.#pendingFailure = failure;
  }

  read(destination: RuntimeSlice<uint8>): [int64, Failure | undefined] {
    if (this.#values.length === 0) {
      return [0, this.#takeFailure()];
    }
    const count = Math.min(destination.length, this.#values.length);
    writeBytes(destination, this.#values.slice(0, count));
    this.#values.splice(0, count);
    return [count, undefined];
  }

  readByte(): [uint8, Failure | undefined] {
    const value = this.#values.shift();
    return value === undefined
      ? [0, this.#takeFailure()]
      : [value, undefined];
  }

  #takeFailure(): Failure | undefined {
    const failure = this.#pendingFailure;
    this.#pendingFailure = undefined;
    return failure;
  }
}

export class CanonicalBufioReader<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReader<Failure>,
> {
  readonly #state = new ReaderBuffer<Failure>();
  readonly #source: Source | undefined;
  readonly #noProgress: Failure;

  constructor(
    source: Source | undefined,
    noProgress: Failure,
  ) {
    this.#source = source;
    this.#noProgress = noProgress;
  }

  static Read<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    receiver: CanonicalBufioReader<Failure, Source> | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]> {
    return requireReader(receiver).Read(destination, recovery);
  }

  static ReadByte<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    receiver: CanonicalBufioReader<Failure, Source> | undefined,
    recovery?: GoRecovery,
  ): Promise<[uint8, Failure | undefined]> {
    return requireReader(receiver).ReadByte(recovery);
  }

  static ReadBytes<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    receiver: CanonicalBufioReader<Failure, Source> | undefined,
    delimiter: uint8,
    recovery?: GoRecovery,
  ): Promise<[RuntimeSlice<uint8>, Failure | undefined]> {
    return requireReader(receiver).ReadBytes(delimiter, recovery);
  }

  async Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]> {
    if (destination.length === 0) {
      return [0, undefined];
    }
    await this.#fill(recovery);
    return this.#state.read(destination);
  }

  async ReadByte(
    recovery?: GoRecovery,
  ): Promise<[uint8, Failure | undefined]> {
    await this.#fill(recovery);
    return this.#state.readByte();
  }

  async ReadBytes(
    delimiter: uint8,
    recovery?: GoRecovery,
  ): Promise<[RuntimeSlice<uint8>, Failure | undefined]> {
    const values: number[] = [];
    for (;;) {
      const [value, failure] = await this.ReadByte(recovery);
      if (failure !== undefined) {
        return [byteSlice(values), failure];
      }
      values.push(value);
      if (value === delimiter) {
        return [byteSlice(values), undefined];
      }
    }
  }

  async #fill(recovery?: GoRecovery): Promise<void> {
    if (!this.#state.shouldFill()) {
      return;
    }
    for (let attempt = 0; attempt < 100; attempt += 1) {
      const target = readBuffer();
      const source = requireSource(this.#source);
      const [count, failure] = await source.Read(target, recovery);
      this.#state.accept(target, count, failure);
      if (count > 0 || failure !== undefined) {
        return;
      }
    }
    this.#state.setFailure(this.#noProgress);
  }
}

export function NewReaderCanonical<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReader<Failure>,
>(
  source: Source | undefined,
  noProgress: Failure | undefined,
): CanonicalBufioReader<Failure, Source> {
  return new CanonicalBufioReader(source, requireNoProgress(noProgress));
}

function readBuffer(): RuntimeSlice<uint8> {
  return RuntimeSlice.make<uint8>(defaultBufferSize, defaultBufferSize, 0);
}

function requireSource<Source>(source: Source | undefined): Source {
  if (source === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return source;
}

function requireNoProgress<Failure>(failure: Failure | undefined): Failure {
  if (failure === undefined) {
    GoPanic.raiseRuntime("gostdlib provider supplied a nil io.ErrNoProgress");
  }
  return failure;
}

function requireReader<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReader<Failure>,
>(
  receiver: CanonicalBufioReader<Failure, Source> | undefined,
): CanonicalBufioReader<Failure, Source> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}
