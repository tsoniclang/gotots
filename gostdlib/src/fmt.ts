import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/runtime/scalars.js";

export interface Stringer extends GoInterfaceValue {
  String(): gostring;
}
