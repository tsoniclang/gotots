import { Read as Read__from_sink } from "../sink/package.js";
import { $initializeState, $state } from "./state.js";
export function $initialize(): void {
    const assignmentValue: int32 = 0;
    $initializeState(assignmentValue);
    {
        $state.Observed = Read__from_sink();
    }
}
export { Run } from "../../../../modules/example.com/package-initialization/api/api.js";
export { $state };
import type { int32 } from "@gotots/runtime/scalars.js";
