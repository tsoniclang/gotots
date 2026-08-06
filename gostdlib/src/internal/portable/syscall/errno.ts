import {
  GoErrorMethodToken,
  type GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";

import type { Errno } from "../../../syscall.js";
import { hostInteger } from "../../host-integer.js";

const errnoType = Object.freeze({ comparable: true });

export class ErrnoError extends GoInterfaceValue implements GoError {
  readonly $go$type = errnoType;
  readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);
  readonly $go$formatString = false;

  constructor(private readonly errno: Errno) {
    super();
  }

  Error(): string {
    return this.errno.Error();
  }

  Is(target: GoError | undefined): boolean {
    return this.errno.Is(target);
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token: object): boolean => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return other instanceof ErrnoError
      && this.errno.value === other.errno.value;
  }

  $go$hash(): number {
    return hostInteger(this.errno.value);
  }

  $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "syscall.Errno";
    }
    const message = this.Error();
    return verb === "q" ? JSON.stringify(message) : message;
  }
}

export function errnoError(errno: Errno): GoError {
  return new ErrnoError(errno);
}
