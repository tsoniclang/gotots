import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { isGoError } from "../../runtime/error.js";

export interface FormattedText {
  readonly text: string;
  readonly wrapped: readonly GoError[];
}

interface Directive {
  readonly flags: string;
  readonly width: number | undefined;
  readonly precision: number | undefined;
  readonly verb: string;
  readonly next: number;
}

export function formatText(
  format: string,
  values: RuntimeSlice<GoInterfaceValue | undefined>,
): FormattedText {
  let text = "";
  let argumentIndex = 0;
  const wrapped: GoError[] = [];
  for (let index = 0; index < format.length;) {
    const marker = format.indexOf("%", index);
    if (marker < 0) {
      text += format.slice(index);
      break;
    }
    text += format.slice(index, marker);
    if (format[marker + 1] === "%") {
      text += "%";
      index = marker + 2;
      continue;
    }
    const parsed = parseDirective(format, marker + 1, values, argumentIndex);
    argumentIndex = parsed.argumentIndex;
    const directive = parsed.directive;
    if (argumentIndex >= values.length) {
      text += `%!${directive.verb}(MISSING)`;
      index = directive.next;
      continue;
    }
    const value = values.get(argumentIndex);
    argumentIndex += 1;
    if (directive.verb === "w" && value !== undefined && isGoError(value)) {
      wrapped.push(value);
    }
    const rendered = formatValue(
      value,
      directive.verb === "w" ? "v" : directive.verb,
      directive.flags,
      directive.precision,
    );
    text += applyWidth(rendered, directive.flags, directive.width, directive.verb);
    index = directive.next;
  }
  return { text, wrapped };
}

export function formatOperands(
  values: RuntimeSlice<GoInterfaceValue | undefined>,
  line: boolean,
): string {
  let result = "";
  let previousString = false;
  for (let index = 0; index < values.length; index += 1) {
    const value = values.get(index);
    const stringValue = value?.$go$formatString ?? false;
    if (index !== 0 && (line || (!stringValue && !previousString))) {
      result += " ";
    }
    result += formatValue(value, "v", "", undefined);
    previousString = stringValue;
  }
  return line ? result + "\n" : result;
}

function parseDirective(
  format: string,
  start: number,
  values: RuntimeSlice<GoInterfaceValue | undefined>,
  initialArgumentIndex: number,
): { readonly directive: Directive; readonly argumentIndex: number } {
  let index = start;
  let argumentIndex = initialArgumentIndex;
  let flags = "";
  while (index < format.length && "#+- 0".includes(format[index] ?? "")) {
    flags += format[index];
    index += 1;
  }
  const widthResult = parseMeasure(format, index, values, argumentIndex);
  index = widthResult.next;
  argumentIndex = widthResult.argumentIndex;
  let precision: number | undefined;
  if (format[index] === ".") {
    const precisionResult = parseMeasure(format, index + 1, values, argumentIndex);
    precision = precisionResult.value ?? 0;
    index = precisionResult.next;
    argumentIndex = precisionResult.argumentIndex;
  }
  const verb = format[index] ?? "?";
  return {
    directive: {
      flags,
      width: widthResult.value,
      precision,
      verb,
      next: Math.min(index + 1, format.length),
    },
    argumentIndex,
  };
}

function parseMeasure(
  format: string,
  start: number,
  values: RuntimeSlice<GoInterfaceValue | undefined>,
  initialArgumentIndex: number,
): {
  readonly value: number | undefined;
  readonly next: number;
  readonly argumentIndex: number;
} {
  if (format[start] === "*") {
    const selected = initialArgumentIndex < values.length
      ? values.get(initialArgumentIndex)
      : undefined;
    const decimal = formatValue(selected, "d", "", undefined);
    const value = Number.parseInt(decimal, 10);
    return {
      value: Number.isNaN(value) ? undefined : value,
      next: start + 1,
      argumentIndex: initialArgumentIndex + 1,
    };
  }
  let index = start;
  while (index < format.length && isDigit(format[index] ?? "")) {
    index += 1;
  }
  if (index === start) {
    return { value: undefined, next: start, argumentIndex: initialArgumentIndex };
  }
  return {
    value: Number.parseInt(format.slice(start, index), 10),
    next: index,
    argumentIndex: initialArgumentIndex,
  };
}

function formatValue(
  value: GoInterfaceValue | undefined,
  verb: string,
  flags: string,
  precision: number | undefined,
): string {
  if (value === undefined) {
    return verb === "v" || verb === "T" ? "<nil>" : `%!${verb}(<nil>)`;
  }
  if (isGoError(value) && (verb === "v" || verb === "s" || verb === "q")) {
    const message = value.Error();
    return verb === "q" ? JSON.stringify(message) : message;
  }
  return value.$go$format(verb, flags, precision);
}

function applyWidth(
  rendered: string,
  flags: string,
  selectedWidth: number | undefined,
  verb: string,
): string {
  let result = rendered;
  if (verb === "X") {
    result = result.toUpperCase();
  }
  if (flags.includes("#") && (verb === "x" || verb === "X")) {
    result = (verb === "X" ? "0X" : "0x") + result;
  }
  let width = selectedWidth;
  let left = flags.includes("-");
  if (width !== undefined && width < 0) {
    width = -width;
    left = true;
  }
  if (width === undefined || result.length >= width) {
    return result;
  }
  const padding = (flags.includes("0") && !left ? "0" : " ").repeat(width - result.length);
  if (left) {
    return result + padding;
  }
  if (padding[0] === "0" && (result[0] === "+" || result[0] === "-")) {
    return result[0] + padding + result.slice(1);
  }
  return padding + result;
}

function isDigit(value: string): boolean {
  return value >= "0" && value <= "9";
}
