import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoString } from "@gotots/runtime/string-value.js";
import type { gostring, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import { ProviderInterfaceValue } from "../portable/io/value.js";

export interface CanonicalError extends GoInterfaceValue {
  Error(recovery?: GoRecovery): gostring;
}

const canonicalBoundaryErrorType = Object.freeze({ comparable: true });

export class CanonicalBoundaryError extends ProviderInterfaceValue
  implements CanonicalError {
  override readonly $go$methods: ReadonlySet<object>;
  readonly #message: GoString;

  constructor(
    message: string,
    contract: readonly object[],
  ) {
    super(canonicalBoundaryErrorType);
    this.#message = GoString.fromText(message);
    this.$go$methods = new Set(contract);
  }

  override $go$format(
    verb: string,
    _flags: string,
    _precision: number | undefined,
  ): string {
    const message = this.#message.text();
    return verb === "q" ? JSON.stringify(message) : message;
  }

  Error(): gostring {
    return this.#message;
  }
}

export interface CanonicalReader<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined];
}

export interface ProviderReaderInterface<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined];
}

export interface CanonicalWriter<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined];
}

export interface ProviderWriterInterface<Failure extends GoInterfaceValue>
  extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined];
}
