import { GoString } from "@gotots/runtime/string-value.js";

export function textValues<Value>(values: readonly (GoString | Value)[]): (string | Value)[] {
  return values.map(value => value instanceof GoString ? value.text() : value);
}
