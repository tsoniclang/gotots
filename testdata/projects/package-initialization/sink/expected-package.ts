import { $initializeState, $state } from "./state.js";
export function $initialize(): void {
    const assignmentValue: int32 = 0;
    $initializeState(assignmentValue);
}
export { Mark, Pair, Read } from "../../../../modules/example.com/package-initialization/sink/sink.js";
export { $state };
import type { int32 } from "@gotots/runtime/scalars.js";
