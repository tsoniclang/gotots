import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { GoPointer } from "@gotots/runtime/pointer.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  Awaitable,
  bool,
  gostring,
  int64,
} from "@gotots/runtime/scalars.js";
import { ProviderError } from "../../runtime/error.js";
import { sliceValues } from "../../runtime/slice.js";

export interface ErrorHandlingValue {
  readonly value: int64;
}

export interface FlagSetValue {
  Usage: (() => Awaitable<void>) | undefined;
}

interface BooleanBinding {
  readonly kind: "boolean";
  readonly pointer: GoPointer<bool, bool>;
}

interface StringBinding {
  readonly kind: "string";
  readonly pointer: GoPointer<gostring, gostring>;
}

interface FlagSetState {
  readonly name: gostring;
  readonly errorHandling: ErrorHandlingValue;
  readonly bindings: Map<string, BooleanBinding | StringBinding>;
}

const states = new WeakMap<FlagSetValue, FlagSetState>();
const continueOnError: ErrorHandlingValue = Object.freeze({ value: 0 });

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

export function booleanFlag(
  receiver: FlagSetValue | undefined,
  name: gostring,
  value: bool,
): GoPointer<bool, bool> {
  const state = requireFlagSet(receiver);
  const pointer = GoPointer.cell<bool, bool>(value);
  add(state, name, {
    kind: "boolean",
    pointer,
  });
  return pointer;
}

export function stringFlag(
  receiver: FlagSetValue | undefined,
  name: gostring,
  value: gostring,
): GoPointer<gostring, gostring> {
  const state = requireFlagSet(receiver);
  const pointer = GoPointer.cell<gostring, gostring>(value);
  add(state, name, {
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
    const argument = values[index] ?? "";
    if (argument === "--") {
      return undefined;
    }
    if (!argument.startsWith("-") || argument === "-") {
      return undefined;
    }
    const spelling = argument.startsWith("--")
      ? argument.slice(2)
      : argument.slice(1);
    const separator = spelling.indexOf("=");
    const name = separator < 0 ? spelling : spelling.slice(0, separator);
    const inlineValue = separator < 0
      ? undefined
      : spelling.slice(separator + 1);
    const binding = state.bindings.get(name);
    if (binding === undefined) {
      if (name === "h" || name === "help") {
        void receiver?.Usage?.();
        return failure(state, "flag: help requested");
      }
      return failure(state, `flag provided but not defined: -${name}`);
    }
    if (binding.kind === "boolean") {
      const text = inlineValue ?? "true";
      const parsed = parseBoolean(text);
      if (parsed === undefined) {
        return failure(state, `invalid value ${text} for flag -${name}`);
      }
      binding.pointer.value = parsed;
      continue;
    }
    const next = inlineValue ?? values[index + 1];
    if (next === undefined) {
      return failure(state, `flag needs an argument: -${name}`);
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
      name: "",
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
  if (state.errorHandling.value === 2) {
    GoPanic.raiseRuntime(message);
  }
  if (state.errorHandling.value === 1) {
    process.exit(2);
  }
  return new ProviderError(
    state.name.length === 0 ? message : `${state.name}: ${message}`,
  );
}
