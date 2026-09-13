import { GoString } from "@gotots/runtime/string-value.js";
import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int64,
} from "@gotots/gostdlib/internal/scalars.js";
import { ProviderError } from "../../runtime/error.js";
import {
  providerPointer,
  type ProviderPointer,
} from "../../runtime/pointer.js";
import { sliceValues } from "../../runtime/slice.js";

export interface ErrorHandlingValue {
  readonly value: int64;
}

export interface FlagSetValue {
  Usage: (() => void) | undefined;
}

interface BooleanBinding {
  readonly kind: "boolean";
  readonly pointer: ProviderPointer<bool>;
}

interface StringBinding {
  readonly kind: "string";
  readonly pointer: ProviderPointer<gostring>;
}

interface FlagSetState {
  readonly name: gostring;
  readonly errorHandling: ErrorHandlingValue;
  readonly bindings: Map<string, BooleanBinding | StringBinding>;
}

const states = new WeakMap<FlagSetValue, FlagSetState>();
const continueOnError: ErrorHandlingValue = Object.freeze({ value: 0n });

export function initializeFlagSet(
  receiver: FlagSetValue,
  name: gostring,
  errorHandling: ErrorHandlingValue,
): void {
  states.set(receiver, {
    name,
    errorHandling,
    bindings: new Map(),
  });
}

export function assignFlagSet(
  target: FlagSetValue,
  source: FlagSetValue,
): void {
  target.Usage = source.Usage;
  const sourceState = requireFlagSet(source);
  states.set(target, {
    name: sourceState.name,
    errorHandling: sourceState.errorHandling,
    bindings: sourceState.bindings,
  });
}

export function booleanFlag(
  receiver: FlagSetValue | undefined,
  name: gostring,
  value: bool,
): ProviderPointer<bool> {
  const state = requireFlagSet(receiver);
  const pointer = providerPointer(value);
  add(state, name.text(), {
    kind: "boolean",
    pointer,
  });
  return pointer;
}

export function stringFlag(
  receiver: FlagSetValue | undefined,
  name: gostring,
  value: gostring,
): ProviderPointer<gostring> {
  const state = requireFlagSet(receiver);
  const pointer = providerPointer(value);
  add(state, name.text(), {
    kind: "string",
    pointer,
  });
  return pointer;
}

export function parseFlags(
  receiver: FlagSetValue | undefined,
  arguments_: RuntimeSlice<gostring>,
): GoError | undefined {
  const state = requireFlagSet(receiver);
  const values = sliceValues(arguments_);
  for (let index = 0; index < values.length; index += 1) {
    const argument = values[index];
    if (argument === undefined) throw new RangeError("flag argument is missing");
    const argumentText = argument.text();
    if (argumentText === "--") {
      return undefined;
    }
    if (!argumentText.startsWith("-") || argumentText === "-") {
      return undefined;
    }
    const spelling = argumentText.startsWith("--")
      ? argument.slice(2)
      : argument.slice(1);
    const separator = spelling.text().indexOf("=");
    const name = separator < 0 ? spelling : spelling.slice(0, separator);
    const inlineValue = separator < 0
      ? undefined
      : spelling.slice(separator + 1);
    const binding = state.bindings.get(name.text());
    if (binding === undefined) {
      if (name.text() === "h" || name.text() === "help") {
        receiver?.Usage?.();
        return failure(state, "flag: help requested");
      }
      return failure(state, `flag provided but not defined: -${name.text()}`);
    }
    if (binding.kind === "boolean") {
      const text = inlineValue?.text() ?? "true";
      const parsed = parseBoolean(text);
      if (parsed === undefined) {
        return failure(state, `invalid value ${text} for flag -${name.text()}`);
      }
      binding.pointer.value = parsed;
      continue;
    }
    const next = inlineValue ?? values[index + 1];
    if (next === undefined) {
      return failure(state, `flag needs an argument: -${name.text()}`);
    }
    if (inlineValue === undefined) {
      index += 1;
    }
    binding.pointer.value = next;
  }
  return undefined;
}

function parseBoolean(value: string): boolean | undefined {
  switch (value) {
    case "1":
    case "t":
    case "T":
    case "true":
    case "TRUE":
    case "True":
      return true;
    case "0":
    case "f":
    case "F":
    case "false":
    case "FALSE":
    case "False":
      return false;
    default:
      return undefined;
  }
}

function requireFlagSet(receiver: FlagSetValue | undefined): FlagSetState {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("flag: nil FlagSet");
  }
  let state = states.get(receiver);
  if (state === undefined) {
    state = {
      name: GoString.empty,
      errorHandling: continueOnError,
      bindings: new Map(),
    };
    states.set(receiver, state);
  }
  return state;
}

function add(
  state: FlagSetState,
  name: string,
  binding: BooleanBinding | StringBinding,
): void {
  if (state.bindings.has(name)) {
    GoPanic.raiseRuntime(`flag redefined: ${name}`);
  }
  state.bindings.set(name, binding);
}

function failure(state: FlagSetState, message: string): GoError {
  if (state.errorHandling.value === 2n) {
    GoPanic.raiseRuntime(message);
  }
  if (state.errorHandling.value === 1n) {
    process.exit(2);
  }
  return ProviderError.fromText(
    state.name.text().length === 0 ? message : `${state.name.text()}: ${message}`,
  );
}
