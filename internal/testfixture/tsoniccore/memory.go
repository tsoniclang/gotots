package tsoniccore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const memoryTypeDeclarations = `
export interface DataLayout { readonly __tsonicDataLayout: "DataLayout"; }
export interface MemoryLayout<T> { readonly __tsonicMemoryLayout: (value: T) => T; }
export interface MemoryFieldLayout<T> { readonly __tsonicMemoryFieldLayout: (value: T) => T; }
export type nativeUint = number;
`

const memoryOperationDeclarations = `
import type { DataLayout, MemoryLayout, MemoryFieldLayout, RawPointer, nativeUint } from "./types.js";
export declare function reinterpretRawPointer<T>(pointer: RawPointer | undefined, layout: MemoryLayout<T>): Pointer<T> | undefined;
export declare function offsetRawPointer<TOffset extends number | bigint>(pointer: RawPointer | undefined, byteOffset: TOffset, dataLayout: DataLayout): RawPointer | undefined;
export declare function rawPointerToAddressInteger(pointer: RawPointer | undefined, dataLayout: DataLayout): nativeUint;
export declare function addressIntegerToRawPointer(address: nativeUint, dataLayout: DataLayout): RawPointer | undefined;
export declare function memoryLayout<T>(dataLayout: DataLayout, byteSize: nativeUint, byteAlignment: nativeUint, stride: nativeUint, ...fields: MemoryFieldLayout<T>[]): MemoryLayout<T>;
export declare function memoryField<T, TField>(select: (value: T) => TField, byteOffset: nativeUint, byteAlignment: nativeUint): MemoryFieldLayout<T>;
export declare function sizeOf<T>(layout: MemoryLayout<T>): nativeUint;
export declare function alignOf<T>(layout: MemoryLayout<T>): nativeUint;
export declare function strideOf<T>(layout: MemoryLayout<T>): nativeUint;
export declare function fieldOffsetOf<T, TField>(layout: MemoryLayout<T>, select: (value: T) => TField): nativeUint;
export declare function keepAlive<T>(value: T): void;
`

func memoryOperationRuntime() string {
	var output strings.Builder
	for _, name := range []string{
		"reinterpretRawPointer", "offsetRawPointer", "rawPointerToAddressInteger", "addressIntegerToRawPointer",
		"memoryLayout", "memoryField", "sizeOf", "alignOf", "strideOf", "fieldOffsetOf", "keepAlive",
	} {
		fmt.Fprintf(&output, "export const %s = () => unsupported(%q);\n", name, name)
	}
	return output.String()
}

func installABIResolution(root string) error {
	directory := filepath.Join(root, "node_modules", "@gotots", "abi")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	files := map[string]string{
		"package.json": `{"type":"module","exports":{"./layout.js":"./layout.js"}}`,
		"layout.d.ts": `import type { DataLayout } from "@tsonic/core/types.js";
export declare const little32: DataLayout;
export declare const little64: DataLayout;
export declare const big32: DataLayout;
export declare const big64: DataLayout;
`,
		"layout.js": "throw new Error(\"resolution-only ABI fixture executed\");\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
			return fmt.Errorf("install ABI resolution fixture %s: %w", name, err)
		}
	}
	return nil
}
