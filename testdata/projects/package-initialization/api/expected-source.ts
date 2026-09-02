import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-initialization/api/state.js";
import { Read as Read__from_sink } from "../../../../packages/example.com/package-initialization/sink/package.js";
export function Run(): int32 {
    return $state.Observed + Read__from_sink() | 0;
}
