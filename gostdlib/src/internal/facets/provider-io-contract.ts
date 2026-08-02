import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  int64,
  uint8,
} from "@gotots/runtime/scalars.js";

import { ProviderInterfaceValue } from "../portable/io/value.js";

export interface CanonicalErrorSync extends GoInterfaceValue {
  Error(recovery?: GoRecovery): gostring;
}

export interface CanonicalErrorAsync extends GoInterfaceValue {
  Error(recovery?: GoRecovery): Promise<gostring>;
}

const canonicalBoundaryErrorType = Object.freeze({ comparable: true });

abstract class CanonicalBoundaryError extends ProviderInterfaceValue {
  override readonly $go$methods: ReadonlySet<object>;

  constructor(
    protected readonly message: string,
    contract: readonly object[],
  ) {
    super(canonicalBoundaryErrorType);
    this.$go$methods = new Set(contract);
  }

  override $go$format(
    verb: string,
    _flags: string,
    _precision: number | undefined,
  ): string {
    return verb === "q" ? JSON.stringify(this.message) : this.message;
  }
}

export class CanonicalBoundaryErrorSync extends CanonicalBoundaryError
  implements CanonicalErrorSync {
  Error(): string {
    return this.message;
  }
}

export class CanonicalBoundaryErrorAsync extends CanonicalBoundaryError
  implements CanonicalErrorAsync {
  async Error(): Promise<string> {
    return this.message;
  }
}

export interface CanonicalReaderSourceSync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined];
}

export interface CanonicalReaderSourceAsync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]>;
}

export interface CanonicalWriterTargetSync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined];
}

export interface CanonicalWriterTargetAsync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Write(
    source: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, Failure | undefined]>;
}
