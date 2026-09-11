import type { Pointer } from "@tsonic/core/types.js";
import { GoPanic } from "./panic.js";
import { addressOf, loadPointer, storePointer } from "@tsonic/core/lang.js";
export type GoStorageRegion<Element> = {
    readonly kind: "indexed";
    readonly values: Element[];
    readonly offset: number;
} | {
    readonly kind: "pointer";
    readonly offset: number | bigint;
    readonly at: (index: number | bigint) => Pointer<Element>;
};
export function goRegionAddress<Element>(region: GoStorageRegion<Element>, index: number | bigint): Pointer<Element> {
    if (region.kind === "indexed") {
        return addressOf<Element>(region.values[region.offset + Number(index)]);
    }
    return region.at(BigInt(region.offset) + BigInt(index));
}
export function goRegionRead<Element>(region: GoStorageRegion<Element>, index: number | bigint): Element {
    if (region.kind === "indexed") {
        return (region.offset + Number(index) in region.values ? region.values[region.offset + Number(index)] : GoPanic.raiseRuntime("dense storage index is absent")) as Element;
    }
    return loadPointer<Element>(region.at(BigInt(region.offset) + BigInt(index)));
}
export function goRegionWrite<Element>(region: GoStorageRegion<Element>, index: number | bigint, value: Element): Element {
    if (region.kind === "indexed") {
        return region.values[region.offset + Number(index)] = value;
    }
    storePointer<Element>(region.at(BigInt(region.offset) + BigInt(index)), value);
    return value;
}
