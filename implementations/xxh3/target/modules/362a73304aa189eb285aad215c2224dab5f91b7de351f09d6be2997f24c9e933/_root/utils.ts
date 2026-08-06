import type { GoUnsafePointer } from "@gotots/runtime/unsafe-pointer.js";

export type str$Storage = {
  p: GoUnsafePointer | undefined;
  l: number;
};
