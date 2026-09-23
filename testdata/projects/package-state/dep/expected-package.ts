import type { Cell$Storage as Cell__from_dep$Storage } from "../../../../modules/example.com/package-state/dep/state.js";
import type { int32 } from "@gotots/runtime/scalars.js";
import { Cell, mark } from "../../../../modules/example.com/package-state/dep/state.js";
import { $initializeState, $state } from "./state.js";
export function $initialize(): void {
    const assignmentValue: int32 = 0;
    const assignmentValue2: int32 = 0;
    const assignmentValue3: int32 = 0;
    const assignmentValue4: Cell__from_dep$Storage = Cell.$zeroStorage();
    const assignmentValue5: Cell__from_dep$Storage = Cell.$zeroStorage();
    const assignmentValue6: int32 = 0;
    const assignmentValue7: int32 = 0;
    const assignmentValue8: int32 = 0;
    $initializeState(assignmentValue, assignmentValue2, assignmentValue3, assignmentValue4, assignmentValue5, assignmentValue6, assignmentValue7, assignmentValue8);
    {
        $state.B = mark(2);
    }
    {
        $state.A = $state.B + mark(1) | 0;
    }
    {
        $state.hidden = mark(3);
    }
    {
        $state.Filled = Cell.$storageOf(Cell.$fromStorage({
            Value: 4
        }));
    }
}
export { Cell, Cell$Storage, Snapshot } from "../../../../modules/example.com/package-state/dep/state.js";
export { $state };
