package tsoniccore

import (
	"fmt"
	"strings"
)

const memoryTypeDeclarations = `
export interface DataLayout { readonly __tsonicDataLayout: "DataLayout"; }
export interface MemoryLayout<T> { readonly __tsonicMemoryLayout: (value: T) => T; }
export interface MemoryFieldLayout<T> { readonly __tsonicMemoryFieldLayout: (value: T) => T; }
export interface MemoryFieldBinding<T> { readonly __tsonicMemoryFieldBinding: (value: T) => T; }
export interface FixedArray<T, TLength extends number | bigint> {
  [index: number]: T;
  readonly length: TLength;
  [Symbol.iterator](): Iterator<T>;
}
export type nativeUint = number;
`

const memoryOperationDeclarations = `
import type { DataLayout, MemoryLayout, MemoryFieldLayout, MemoryFieldBinding, FixedArray, RawPointer, nativeUint } from "./types.js";
export declare function viewPointer<F, T>(pointer: Pointer<F>, read: () => T, write: (value: T) => void): Pointer<T>;
export declare function viewPointer<F, T>(pointer: Pointer<F> | undefined, read: () => T, write: (value: T) => void): Pointer<T> | undefined;
export declare function bindMemoryField<T, TField>(field: MemoryFieldLayout<T>, pointer: Pointer<TField>): MemoryFieldBinding<T>;
export declare function bindMemoryRecord<T>(layout: MemoryLayout<T>, ...fields: MemoryFieldBinding<T>[]): T;
export declare function reinterpretRawPointer<T>(pointer: RawPointer | undefined, layout: MemoryLayout<T>): Pointer<T> | undefined;
export declare function offsetRawPointer<TOffset extends number | bigint>(pointer: RawPointer | undefined, byteOffset: TOffset, dataLayout: DataLayout): RawPointer | undefined;
export declare function rawPointerToAddressInteger<TAddress extends number | bigint>(pointer: RawPointer | undefined, dataLayout: DataLayout): TAddress;
export declare function addressIntegerToRawPointer<TAddress extends number | bigint>(address: TAddress, dataLayout: DataLayout): RawPointer | undefined;
export declare function memoryLayout<T>(dataLayout: DataLayout, byteSize: nativeUint, byteAlignment: nativeUint, stride: nativeUint, ...fields: MemoryFieldLayout<T>[]): MemoryLayout<T>;
export declare function memoryArrayLayout<T, TLength extends number | bigint>(dataLayout: DataLayout, byteSize: nativeUint, byteAlignment: nativeUint, stride: nativeUint, elementLayout: MemoryLayout<T>, length: TLength): MemoryLayout<FixedArray<T, TLength>>;
export declare function memoryField<T, TField>(select: (value: T) => TField, byteOffset: nativeUint, byteAlignment: nativeUint, fieldLayout: MemoryLayout<TField>): MemoryFieldLayout<T>;
export declare function sizeOf<T>(layout: MemoryLayout<T>): nativeUint;
export declare function alignOf<T>(layout: MemoryLayout<T>): nativeUint;
export declare function strideOf<T>(layout: MemoryLayout<T>): nativeUint;
export declare function fieldOffsetOf<T, TField>(layout: MemoryLayout<T>, select: (value: T) => TField): nativeUint;
export declare function keepAlive<T>(value: T): void;
export declare function struct<T>(shape: T): T;
export declare function field<T>(): T;
export declare function defaultValue<T>(): T;
`

func memoryOperationRuntime() string {
	var output strings.Builder
	output.WriteString("export function struct(shape) { return shape; }\nexport function field() { return undefined; }\n")
	for _, name := range []string{
		"reinterpretRawPointer", "offsetRawPointer", "rawPointerToAddressInteger", "addressIntegerToRawPointer",
		"memoryLayout", "memoryArrayLayout", "memoryField", "sizeOf", "alignOf", "strideOf", "fieldOffsetOf", "keepAlive", "defaultValue",
		"viewPointer", "bindMemoryField", "bindMemoryRecord",
	} {
		fmt.Fprintf(&output, "export const %s = () => unsupported(%q);\n", name, name)
	}
	return output.String()
}

func installABIResolution(root string) error {
	files := map[string]string{
		"package.json": `{
  "name": "@gotots/abi",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "files": ["layout.d.ts", "layout.js"],
  "peerDependencies": {"@tsonic/core": "0.0.0"},
  "exports": {"./layout.js": "./layout.js"}
}`,
		"layout.d.ts": `import type { DataLayout } from "@tsonic/core/types.js";
export declare const little32: DataLayout;
export declare const little64: DataLayout;
export declare const big32: DataLayout;
export declare const big64: DataLayout;
`,
		"layout.js": "throw new Error(\"resolution-only ABI fixture executed\");\n",
	}
	return installPackage(root, "@gotots", "abi", files)
}
