import type { GoError } from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { ProviderPointer } from "./internal/runtime/pointer.js";
import type {
  bool,
  gostring,
  int64,
} from "@gotots/gostdlib/internal/scalars.js";
import {
  booleanFlag,
  initializeFlagSet,
  parseFlags,
  stringFlag,
} from "./internal/node/flag/flag-set.js";

export class ErrorHandling {
  constructor(public readonly value: int64) {}
}

export const ContinueOnError = new ErrorHandling(0n);

export class FlagSet {
  Usage: (() => void) | undefined = undefined;

  static Bool(
    receiver: FlagSet | undefined,
    name: gostring,
    value: bool,
    usage: gostring,
  ): ProviderPointer<bool> | undefined {
    void usage;
    return booleanFlag(receiver, name, value);
  }

  static Parse(
    receiver: FlagSet | undefined,
    args: RuntimeSlice<gostring>,
  ): GoError | undefined {
    return parseFlags(receiver, args);
  }

  static String(
    receiver: FlagSet | undefined,
    name: gostring,
    value: gostring,
    usage: gostring,
  ): ProviderPointer<gostring> | undefined {
    void usage;
    return stringFlag(receiver, name, value);
  }
}

export function NewFlagSet(
  name: gostring,
  errorHandling: ErrorHandling,
): FlagSet | undefined {
  const set = new FlagSet();
  initializeFlagSet(set, name, errorHandling);
  return set;
}
