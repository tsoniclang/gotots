import { Snapshot as Snapshot__from_dep } from "../dep/package.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.Start = 0;
    {
        $state.Start = Snapshot__from_dep();
    }
}
export { Run } from "../../../modules/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/api/api.js";
export { $state };
