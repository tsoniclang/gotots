import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  Base64EncoderState,
  Encoding,
  runBase64EncoderAsync,
  runBase64EncoderSync,
} from "../portable/encoding/base64.js";
import { ProviderInterfaceValue } from "../portable/io/value.js";
import type {
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export interface CanonicalWriteCloserSync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined];
  Close(recovery?: GoRecovery): Failure | undefined;
}

export interface CanonicalWriteCloserAsync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]>;
  Close(recovery?: GoRecovery): Promise<Failure | undefined>;
}

export interface CanonicalWriteCloserSyncWriteAsyncClose<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined];
  Close(recovery?: GoRecovery): Promise<Failure | undefined>;
}

const canonicalBase64EncoderType = Object.freeze({ comparable: true });

class CanonicalBase64EncoderSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
> extends ProviderInterfaceValue implements CanonicalWriteCloserSync<Failure> {
  override readonly $go$methods: ReadonlySet<object>;
  readonly #state: Base64EncoderState<Failure>;

  constructor(
    encoding: Encoding | undefined,
    private readonly target: Target | undefined,
    contract: readonly object[],
  ) {
    super(canonicalBase64EncoderType);
    this.#state = new Base64EncoderState(encoding);
    this.$go$methods = new Set(contract);
  }

  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined] {
    return runBase64EncoderSync(
      this.#state.beginWrite(source),
      (output) => requireTarget(this.target).Write(output, recovery),
    );
  }

  Close(recovery?: GoRecovery): Failure | undefined {
    return runBase64EncoderSync(
      this.#state.beginClose(),
      (output) => requireTarget(this.target).Write(output, recovery),
    );
  }
}

class CanonicalBase64EncoderAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
> extends ProviderInterfaceValue implements CanonicalWriteCloserAsync<Failure> {
  override readonly $go$methods: ReadonlySet<object>;
  readonly #state: Base64EncoderState<Failure>;

  constructor(
    encoding: Encoding | undefined,
    private readonly target: Target | undefined,
    contract: readonly object[],
  ) {
    super(canonicalBase64EncoderType);
    this.#state = new Base64EncoderState(encoding);
    this.$go$methods = new Set(contract);
  }

  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]> {
    return runBase64EncoderAsync(
      this.#state.beginWrite(source),
      (output) => requireTarget(this.target).Write(output, recovery),
    );
  }

  Close(recovery?: GoRecovery): Promise<Failure | undefined> {
    return runBase64EncoderAsync(
      this.#state.beginClose(),
      (output) => requireTarget(this.target).Write(output, recovery),
    );
  }
}

class CanonicalBase64EncoderSyncWriteAsyncClose<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
> extends ProviderInterfaceValue
  implements CanonicalWriteCloserSyncWriteAsyncClose<Failure> {
  override readonly $go$methods: ReadonlySet<object>;
  readonly #state: Base64EncoderState<Failure>;

  constructor(
    encoding: Encoding | undefined,
    private readonly target: Target | undefined,
    contract: readonly object[],
  ) {
    super(canonicalBase64EncoderType);
    this.#state = new Base64EncoderState(encoding);
    this.$go$methods = new Set(contract);
  }

  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined] {
    return runBase64EncoderSync(
      this.#state.beginWrite(source),
      (output) => requireTarget(this.target).Write(output, recovery),
    );
  }

  async Close(recovery?: GoRecovery): Promise<Failure | undefined> {
    return runBase64EncoderSync(
      this.#state.beginClose(),
      (output) => requireTarget(this.target).Write(output, recovery),
    );
  }
}

export function Base64NewEncoderCanonicalSync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  encoding: Encoding | undefined,
  target: Target | undefined,
  writeCloserContract: readonly object[],
): CanonicalWriteCloserSync<Failure> {
  return new CanonicalBase64EncoderSync(
    encoding,
    target,
    writeCloserContract,
  );
}

export function Base64NewEncoderCanonicalAsync<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetAsync<Failure>,
>(
  encoding: Encoding | undefined,
  target: Target | undefined,
  writeCloserContract: readonly object[],
): CanonicalWriteCloserAsync<Failure> {
  return new CanonicalBase64EncoderAsync(
    encoding,
    target,
    writeCloserContract,
  );
}

export function Base64NewEncoderCanonicalSyncWriteAsyncClose<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriterTargetSync<Failure>,
>(
  encoding: Encoding | undefined,
  target: Target | undefined,
  writeCloserContract: readonly object[],
): CanonicalWriteCloserSyncWriteAsyncClose<Failure> {
  return new CanonicalBase64EncoderSyncWriteAsyncClose(
    encoding,
    target,
    writeCloserContract,
  );
}

function requireTarget<Target>(target: Target | undefined): Target {
  if (target === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return target;
}
