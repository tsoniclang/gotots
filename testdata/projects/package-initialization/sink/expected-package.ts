import { $state } from "./state.js";
export function $initialize(): void {
    $state.Count = 0;
}
export { Mark, Pair, Read } from "../../../../modules/example.com/package-initialization/sink/sink.js";
export { $state };
