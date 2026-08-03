import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic, type GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  Base64EncoderState,
  Encoding,
  runBase64EncoderAsync,
} from "../portable/encoding/base64.js";
import { ProviderInterfaceValue } from "../portable/io/value.js";
import type { CanonicalWriter } from "./provider-io-contract.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalWriter,
} from "./provider-io-contract.js";

export interface CanonicalWriteCloser<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, Failure | undefined]>;
  Close(recovery?: GoRecovery): Awaitable<Failure | undefined>;
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
