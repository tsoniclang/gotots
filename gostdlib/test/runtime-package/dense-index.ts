import { GoPanic } from "./panic.js";
export class GoDenseIndex {
    public static get<T>(values: readonly T[], index: number): T {
        const value = values[index];
        if (!(index in values))
            GoPanic.raiseRuntime("dense storage index is absent");
        return value as T;
    }
}
