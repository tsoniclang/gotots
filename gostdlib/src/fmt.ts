import { toHostBytes } from "./internal/portable/utf8/codec.js";
import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoString } from "@gotots/runtime/string-value.js";
import type { gostring, int } from "@gotots/gostdlib/internal/scalars.js";

import type { Writer } from "./io.js";
import { File, state as osState } from "./os.js";
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
  const formatted = formatText(format.text(), arguments_);
  if (formatted.wrapped.length === 1) {
    return new MessageWrappedError(formatted.text, formatted.wrapped[0]!);
  }
  if (formatted.wrapped.length > 1) {
    return new MessageWrappedErrors(formatted.text, formatted.wrapped);
  }
  return ProviderError.fromText(formatted.text);
}

export function Fprint(
  writer: Writer | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, GoError | undefined] {
  return write(writer, formatOperands(arguments_, false));
}

export function Fprintf(
  writer: Writer | undefined,
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, GoError | undefined] {
  return write(writer, formatText(format.text(), arguments_).text);
}

export function Fprintln(
  writer: Writer | undefined,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, GoError | undefined] {
  return write(writer, formatOperands(arguments_, true));
}

export function Println(
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): [int, GoError | undefined] {
  const text = formatOperands(arguments_, true);
  return File.Write(osState.Stdout, byteSlice(toHostBytes(GoString.fromText(text))));
}

export function Sprint(
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): gostring {
  return GoString.fromText(formatOperands(arguments_, false));
}

export function Sprintf(
  format: gostring,
  arguments_: RuntimeSlice<GoInterfaceValue | undefined>,
): gostring {
  return GoString.fromText(formatText(format.text(), arguments_).text);
}

function write(
  writer: Writer | undefined,
  text: string,
): [int, GoError | undefined] {
  if (writer === undefined) {
    return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return writer.Write(byteSlice(toHostBytes(GoString.fromText(text))));
}
