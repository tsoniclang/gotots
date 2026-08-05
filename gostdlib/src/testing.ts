import type { bool } from "@gotots/gostdlib/internal/scalars.js";
import { isNodeTest } from "./internal/node/runtime/process.js";

export function Testing(): bool {
  return isNodeTest();
}
