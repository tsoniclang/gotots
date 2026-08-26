import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic, type GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import {
  Base64EncoderState,
  Encoding,
  encodingRepresentationAssign,
  encodingRepresentationCopy,
  runBase64EncoderAsync,
  runBase64EncoderSync,
} from "../portable/encoding/base64.js";
import { ProviderInterfaceValue } from "../portable/io/value.js";
import type {
  CanonicalWriter,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalWriter,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export class Base64EncodingOperations {
  static $copy(source: Encoding): Encoding {
    return encodingRepresentationCopy(source);
  }

  static $assign(target: Encoding, source: Encoding): void {
    encodingRepresentationAssign(target, source);
  }
}

export interface CanonicalWriteCloser<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int, Failure | undefined]>;
  Close(recovery?: GoRecovery): Awaitable<Failure | undefined>;
}

export interface ProviderWriteCloser<Failure extends ProviderErrorInterface>
  extends ProviderWriterInterface<Failure> {
  Close(recovery?: GoRecovery): Failure | undefined;
}

const canonicalBase64EncoderType = Object.freeze({ comparable: true });

class CanonicalBase64Encoder<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
> extends ProviderInterfaceValue implements CanonicalWriteCloser<Failure> {
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
  ): Promise<[int, Failure | undefined]> {
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

const directBase64EncoderType = Object.freeze({ comparable: true });

class DirectBase64Encoder<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
> extends ProviderInterfaceValue implements ProviderWriteCloser<Failure> {
  override readonly $go$methods: ReadonlySet<object>;
  readonly #state: Base64EncoderState<Failure>;

  constructor(
    encoding: Encoding | undefined,
    private readonly target: Target | undefined,
    contract: readonly object[],
  ) {
    super(directBase64EncoderType);
    this.#state = new Base64EncoderState(encoding);
    this.$go$methods = new Set(contract);
  }

  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined] {
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

export function Base64NewEncoderDirect<
  Failure extends ProviderErrorInterface,
  Target extends ProviderWriterInterface<Failure>,
>(
  encoding: Encoding | undefined,
  target: Target | undefined,
  writeCloserContract: InterfaceContract,
): ProviderWriteCloser<Failure> {
  return new DirectBase64Encoder(
    encoding,
    target,
    writeCloserContract,
  );
}

export function Base64NewEncoderCanonical<
  Failure extends GoInterfaceValue,
  Target extends CanonicalWriter<Failure>,
>(
  encoding: Encoding | undefined,
  target: Target | undefined,
  writeCloserContract: InterfaceContract,
): CanonicalWriteCloser<Failure> {
  return new CanonicalBase64Encoder(
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
