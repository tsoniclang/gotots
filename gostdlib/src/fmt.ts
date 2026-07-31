import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64 } from "@gotots/runtime/scalars.js";

import type { Writer } from "./io.js";
import { MessageWrappedError, MessageWrappedErrors } from "./internal/portable/errors/tree.js";
import { formatOperands, formatText } from "./internal/portable/fmt/format.js";
import { byteSlice } from "./internal/runtime/slice.js";
import { ProviderError } from "./internal/runtime/error.js";

export interface Stringer extends GoInterfaceValue {
  String(): gostring;
}

export function Errorf(
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): GoError {
  const formatted = formatText(format, arguments_);
  if (formatted.wrapped.length === 1) {
    return new MessageWrappedError(formatted.text, formatted.wrapped[0]!);
  }
  if (formatted.wrapped.length > 1) {
    return new MessageWrappedErrors(formatted.text, formatted.wrapped);
  }
  return new ProviderError(formatted.text);
}

export function Fprint(
  writer: Writer | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int64, GoError | undefined] {
  return write(writer, formatOperands(arguments_, false));
}

export function Fprintf(
  writer: Writer | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int64, GoError | undefined] {
  return write(writer, formatText(format, arguments_).text);
}

export function Fprintln(
  writer: Writer | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int64, GoError | undefined] {
  return write(writer, formatOperands(arguments_, true));
}

export function Sprint(
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): gostring {
  return formatOperands(arguments_, false);
}

export function Sprintf(
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): gostring {
  return formatText(format, arguments_).text;
}

function write(
  writer: Writer | undefined,
  text: string,
): [int64, GoError | undefined] {
  if (writer === undefined) {
    return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return writer.Write(byteSlice(new TextEncoder().encode(text)));
}
