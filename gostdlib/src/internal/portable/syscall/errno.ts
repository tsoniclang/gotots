import {
  GoErrorMethodToken,
  type GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";

import type { Errno } from "../../../syscall.js";

const errnoType = Object.freeze({});

class ErrnoError extends GoInterfaceValue implements GoError {
  readonly $go$type: object = errnoType;
  readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

  constructor(private readonly errno: Errno) {
    super();
  }

  Error(): string {
    return this.errno.Error();
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token: object): boolean => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return other instanceof ErrnoError
      && this.errno.value === other.errno.value;
  }

  $go$hash(): number {
    return this.errno.value;
  }
}

export function errnoError(errno: Errno): GoError {
  return new ErrnoError(errno);
}
