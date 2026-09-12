import type { GoStorageRegion } from "./memory-view.js";
import type { Pointer, uint8 } from "@tsonic/core/types.js";
import { goRegionAddress, goRegionRead } from "./memory-view.js";
import { GoPanic } from "./panic.js";
import { addressOf } from "@tsonic/core/lang.js";
export class GoStringTextBacking {
    private storage: uint8[] | undefined = void 0;
    constructor(private readonly value: string) {
    }
    read(index: number | bigint): uint8 {
        return this.value.charCodeAt(Number(index)) as uint8;
    }
    text(offset: number | bigint, length: number | bigint): string {
        const start = Number(offset);
        return this.value.slice(start, start + Number(length));
    }
    address(index: number | bigint): Pointer<uint8> | undefined {
        if (index >= this.value.length)
            return void 0;
        const storage = this.storage ?? (this.storage = Array.from(this.value, (character: string): uint8 => character.charCodeAt(0) as uint8));
        return addressOf<uint8>(storage[Number(index)]);
    }
    declare private readonly then?: never;
}
export class GoStringPointerBacking {
    constructor(private readonly region: GoStorageRegion<uint8>) {
    }
    read(index: number | bigint): uint8 {
        return goRegionRead<uint8>(this.region, index);
    }
    address(index: number | bigint): Pointer<uint8> {
        return goRegionAddress<uint8>(this.region, index);
    }
    text(offset: number | bigint, length: number | bigint): string {
        const start = BigInt(offset);
        const numericStart = Number(start);
        const numericLength = Number(length);
        let text: string = "";
        if (Number.isSafeInteger(numericStart) && (Number.isSafeInteger(numericLength) && Number.isSafeInteger(numericStart + numericLength))) {
            const chunk: uint8[] = [];
            for (let index = 0; index < numericLength; index++) {
                chunk.push(goRegionRead<uint8>(this.region, numericStart + index));
                if (chunk.length === 4096) {
                    text += globalThis.String.fromCharCode(...chunk);
                    chunk.length = 0;
                }
            }
            text += globalThis.String.fromCharCode(...chunk);
            return text;
        }
        for (let index = 0n; index < length; index++) {
            text += globalThis.String.fromCharCode(goRegionRead<uint8>(this.region, start + index));
        }
        return text;
    }
    declare private readonly then?: never;
}
export class GoString {
    static readonly empty: GoString = new GoString(new GoStringTextBacking(""), 0, 0);
    private constructor(private readonly backing: GoStringTextBacking | GoStringPointerBacking, private readonly offset: number | bigint, private readonly count: number | bigint) {
    }
    static fromText(value: string): GoString {
        if (value.length === 0)
            return GoString.empty;
        return new GoString(new GoStringTextBacking(value), 0, value.length);
    }
    static fromRegion(region: GoStorageRegion<uint8> | undefined, length: number | bigint): GoString {
        if (length < 0)
            GoPanic.raiseRuntime("unsafe string length is negative");
        if (region === void 0) {
            if (length != 0)
                GoPanic.raiseRuntime("unsafe string on nil pointer");
            return GoString.fromText("");
        }
        return new GoString(new GoStringPointerBacking(region), 0, length);
    }
    sourceLength(): number | bigint {
        return this.count;
    }
    text(): string {
        return this.backing.text(this.offset, this.count);
    }
    data(): Pointer<uint8> | undefined {
        return this.backing.address(this.offset);
    }
    read(index: number | bigint): uint8 {
        if (index < 0 || index >= this.count)
            GoPanic.raiseRuntime("Go string index out of range");
        return this.backing.read(typeof this.offset === "number" && typeof index === "number" ? this.offset + index : BigInt(this.offset) + BigInt(index));
    }
    slice(low: number | bigint, high?: number | bigint): GoString {
        const start = low;
        const end = high ?? this.count;
        if (start < 0 || (end < start || end > this.count))
            GoPanic.raiseRuntime("Go string slice bounds out of range");
        return new GoString(this.backing, typeof this.offset === "number" && typeof start === "number" ? this.offset + start : BigInt(this.offset) + BigInt(start), typeof end === "number" && typeof start === "number" ? end - start : BigInt(end) - BigInt(start));
    }
    declare private readonly then?: never;
}
