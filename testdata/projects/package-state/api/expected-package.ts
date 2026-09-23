import { Snapshot as Snapshot__from_dep } from "../dep/package.js";
import { $initializeState, $state } from "./state.js";
export function $initialize(): void {
    const assignmentValue: int32 = 0;
    $initializeState(assignmentValue);
    {
        $state.Start = Snapshot__from_dep();
    }
}
export { Run } from "../../../../modules/example.com/package-state/api/api.js";
export { $state };
import type { int32 } from "@gotots/runtime/scalars.js";
