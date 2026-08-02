import { Cell, mark } from "../../../modules/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/dep/state.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.A = 0;
    $state.B = 0;
    $state.Dormant = 0;
    $state.Empty = Cell.$storageOf(Cell.$zero());
    $state.Filled = Cell.$storageOf(Cell.$zero());
    $state.Trace = 0;
    $state.__go___proto__ = 0;
    $state.hidden = 0;
    $state.B = mark(2);
    $state.A = $state.B + mark(1);
    $state.hidden = mark(3);
    $state.Filled = Cell.$storageOf(Cell.$make(4));
}
export { Cell, Snapshot } from "../../../modules/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/dep/state.js";
export { $state };
