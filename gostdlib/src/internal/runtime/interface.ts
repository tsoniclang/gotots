import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";

export function goInterfaceEqual(
  left: GoInterfaceValue | undefined,
  right: GoInterfaceValue | undefined,
): boolean {
  return left === undefined
    ? right === undefined
    : right !== undefined && left.$go$equal(right);
}
