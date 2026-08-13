import { Snapshot as Snapshot__from_dep } from "../dep/package.js";
import { $state } from "./state.js";
export function $initialize(): void {
    $state.Start = 0;
    {
        $state.Start = Snapshot__from_dep();
    }
}
export { Run } from "../../../../modules/example.com/package-state/api/api.js";
export { $state };
