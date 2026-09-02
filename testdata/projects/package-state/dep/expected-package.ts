import type { int32 } from "@gotots/runtime/scalars.js";
import { Cell, mark } from "../../../../modules/example.com/package-state/dep/state.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.A = 0;
    $state.B = 0;
    $state.Dormant = 0;
    $state.Empty = Cell.$zeroStorage();
    $state.Filled = Cell.$zeroStorage();
    $state.Trace = 0;
    $state.__go___proto__ = 0;
    $state.hidden = 0;
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
