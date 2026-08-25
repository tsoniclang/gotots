import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
} from "../host-integer.js";

import { byteSlice, writeBytes } from "../runtime/slice.js";
import type {
  CanonicalReader,
  ProviderReaderInterface,
} from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type {
  CanonicalError,
  CanonicalReader,
  ProviderReaderInterface,
} from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

const defaultBufferSize = 4096;

class ReaderBuffer<Failure> {
  readonly #values: number[];
  #read: number;
  #written: number;
  #pendingFailure: Failure | undefined;

  constructor(
    values: number[] = [],
    read = 0,
    written = 0,
    pendingFailure: Failure | undefined = undefined,
  ) {
    this.#values = values;
    this.#read = read;
    this.#written = written;
    this.#pendingFailure = pendingFailure;
  }

  copy(): ReaderBuffer<Failure> {
    return new ReaderBuffer(
      this.#values,
      this.#read,
      this.#written,
      this.#pendingFailure,
    );
  }

  shouldFill(): boolean {
    return this.#read === this.#written && this.#pendingFailure === undefined;
  }

  accept(
    source: RuntimeSlice<uint8>,
    count: int,
    failure: Failure | undefined,
  ): void {
    const hostCount = hostInteger(count);
    this.#read = 0;
    this.#written = hostCount;
    for (let index = 0; index < hostCount; index += 1) {
      this.#values[index] = source.get(index);
    }
    this.#pendingFailure = failure;
  }

  setFailure(failure: Failure): void {
    this.#pendingFailure = failure;
  }

  read(destination: RuntimeSlice<uint8>): [int, Failure | undefined] {
    if (this.#read === this.#written) {
      return [0n, this.#takeFailure()];
    }
    const count = Math.min(destination.length, this.#written - this.#read);
    writeBytes(destination, this.#values.slice(this.#read, this.#read + count));
    this.#read += count;
    return [integerFromHost(count), undefined];
  }

  readByte(): [uint8, Failure | undefined] {
    if (this.#read === this.#written) {
      return [0, this.#takeFailure()];
    }
    const value = this.#values[this.#read] ?? 0;
    this.#read += 1;
    return [value, undefined];
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
  #state = new ReaderBuffer<Failure>();
  #source: Source | undefined;
  #noProgress: Failure;

  constructor(
    source: Source | undefined,
    noProgress: Failure,
  ) {
    this.#source = source;
    this.#noProgress = noProgress;
  }

  static $copy<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    source: CanonicalBufioReader<Failure, Source>,
  ): CanonicalBufioReader<Failure, Source> {
    const target = new CanonicalBufioReader(source.#source, source.#noProgress);
    target.#state = source.#state.copy();
    return target;
  }

  static $assign<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    target: CanonicalBufioReader<Failure, Source>,
    source: CanonicalBufioReader<Failure, Source>,
  ): void {
    const state = source.#state.copy();
    const providerSource = source.#source;
    const noProgress = source.#noProgress;
    target.#state = state;
    target.#source = providerSource;
    target.#noProgress = noProgress;
  }

  static Read<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    receiver: CanonicalBufioReader<Failure, Source> | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int, Failure | undefined]> {
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
  ): Promise<[int, Failure | undefined]> {
    if (destination.length === 0) {
      return [0n, undefined];
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
      if (count > 0n || failure !== undefined) {
        return;
      }
    }
    this.#state.setFailure(this.#noProgress);
  }
}

export class DirectBufioReader<
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
> {
  #state = new ReaderBuffer<Failure>();
  #source: Source | undefined;
  #noProgress: Failure;

  constructor(source: Source | undefined, noProgress: Failure) {
    this.#source = source;
    this.#noProgress = noProgress;
  }

  static $copy<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    source: DirectBufioReader<Failure, Source>,
  ): DirectBufioReader<Failure, Source> {
    const target = new DirectBufioReader(source.#source, source.#noProgress);
    target.#state = source.#state.copy();
    return target;
  }

  static $assign<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    target: DirectBufioReader<Failure, Source>,
    source: DirectBufioReader<Failure, Source>,
  ): void {
    target.#state = source.#state.copy();
    target.#source = source.#source;
    target.#noProgress = source.#noProgress;
  }

  static Read<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    receiver: DirectBufioReader<Failure, Source> | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined] {
    return requireDirectReader(receiver).Read(destination, recovery);
  }

  static ReadByte<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    receiver: DirectBufioReader<Failure, Source> | undefined,
    recovery?: GoRecovery,
  ): [uint8, Failure | undefined] {
    return requireDirectReader(receiver).ReadByte(recovery);
  }

  static ReadBytes<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    receiver: DirectBufioReader<Failure, Source> | undefined,
    delimiter: uint8,
    recovery?: GoRecovery,
  ): [RuntimeSlice<uint8>, Failure | undefined] {
    return requireDirectReader(receiver).ReadBytes(delimiter, recovery);
  }

  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined] {
    if (destination.length === 0) {
      return [0n, undefined];
    }
    this.#fill(recovery);
    return this.#state.read(destination);
  }

  ReadByte(recovery?: GoRecovery): [uint8, Failure | undefined] {
    this.#fill(recovery);
    return this.#state.readByte();
  }

  ReadBytes(
    delimiter: uint8,
    recovery?: GoRecovery,
  ): [RuntimeSlice<uint8>, Failure | undefined] {
    const values: number[] = [];
    for (;;) {
      const [value, failure] = this.ReadByte(recovery);
      if (failure !== undefined) {
        return [byteSlice(values), failure];
      }
      values.push(value);
      if (value === delimiter) {
        return [byteSlice(values), undefined];
      }
    }
  }

  #fill(recovery?: GoRecovery): void {
    if (!this.#state.shouldFill()) {
      return;
    }
    for (let attempt = 0; attempt < 100; attempt += 1) {
      const target = readBuffer();
      const source = requireSource(this.#source);
      const [count, failure] = source.Read(target, recovery);
      this.#state.accept(target, count, failure);
      if (count > 0n || failure !== undefined) {
        return;
      }
    }
    this.#state.setFailure(this.#noProgress);
  }
}

export class BufioReaderOperations {
  static $copy<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    source: CanonicalBufioReader<Failure, Source>,
  ): CanonicalBufioReader<Failure, Source> {
    return CanonicalBufioReader.$copy(source);
  }

  static $assign<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    target: CanonicalBufioReader<Failure, Source>,
    source: CanonicalBufioReader<Failure, Source>,
  ): void {
    CanonicalBufioReader.$assign(target, source);
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

export function NewReaderDirect<
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
>(
  source: Source | undefined,
  noProgress: Failure | undefined,
): DirectBufioReader<Failure, Source> {
  return new DirectBufioReader(source, requireNoProgress(noProgress));
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

function requireDirectReader<
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
>(
  receiver: DirectBufioReader<Failure, Source> | undefined,
): DirectBufioReader<Failure, Source> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}
