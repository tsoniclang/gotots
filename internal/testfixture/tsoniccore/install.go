package tsoniccore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corecontract "github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
)

func InstallResolutionOnly(root string) error {
	if root == "" {
		return fmt.Errorf("install Tsonic core resolution fixture: root is absent")
	}
	module := filepath.Join(root, "node_modules", "@tsonic", "core")
	if err := os.MkdirAll(module, 0o755); err != nil {
		return fmt.Errorf("install Tsonic core resolution fixture: %w", err)
	}
	files, err := fixtureFiles()
	if err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(module, name), []byte(content), 0o600); err != nil {
			return fmt.Errorf("install Tsonic core resolution fixture %s: %w", name, err)
		}
	}
	return nil
}

func fixtureFiles() (map[string]string, error) {
	addressOf, err := exportName(corecontract.SymbolAddressOf)
	if err != nil {
		return nil, err
	}
	allocatePointer, err := exportName(corecontract.SymbolAllocatePointer)
	if err != nil {
		return nil, err
	}
	loadPointer, err := exportName(corecontract.SymbolLoadPointer)
	if err != nil {
		return nil, err
	}
	storePointer, err := exportName(corecontract.SymbolStorePointer)
	if err != nil {
		return nil, err
	}
	equalPointer, err := exportName(corecontract.SymbolEqualPointer)
	if err != nil {
		return nil, err
	}
	hashPointer, err := exportName(corecontract.SymbolHashPointer)
	if err != nil {
		return nil, err
	}
	projectPointer, err := exportName(corecontract.SymbolProjectPointer)
	if err != nil {
		return nil, err
	}
	bindPointer, err := exportName(corecontract.SymbolBindPointer)
	if err != nil {
		return nil, err
	}
	bindRawPointer, err := exportName(corecontract.SymbolBindRawPointer)
	if err != nil {
		return nil, err
	}
	equalRawPointer, err := exportName(corecontract.SymbolEqualRawPointer)
	if err != nil {
		return nil, err
	}
	hashRawPointer, err := exportName(corecontract.SymbolHashRawPointer)
	if err != nil {
		return nil, err
	}
	attribute, err := exportName(corecontract.SymbolAttribute)
	if err != nil {
		return nil, err
	}
	primitiveTypes, err := primitiveTypeDeclarations()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"package.json": `{
  "type": "module",
  "exports": {
    "./lang.js": "./lang.js",
    "./types.js": "./types.js"
  }
}
`,
		"types.d.ts": `declare const pointerBrand: unique symbol;
export interface Pointer<T> {
  readonly [pointerBrand]: (value: T) => T;
}
declare const rawPointerBrand: unique symbol;
export interface RawPointer {
  readonly [rawPointerBrand]: void;
}
` + primitiveTypes,
		"types.js": "export {};\n",
		"lang.d.ts": fmt.Sprintf(`import type { Pointer } from "./types.js";
export declare function %[1]s<T>(storage: T | undefined): Pointer<T>;
export declare function %[2]s<T>(initial: T): Pointer<T>;
export declare function %[3]s<T>(pointer: Pointer<T>): T;
export declare function %[4]s<T>(pointer: Pointer<T>, value: T): void;
export declare function %[5]s<T>(left: Pointer<T> | undefined, right: Pointer<T> | undefined): boolean;
export declare function %[6]s<T>(pointer: Pointer<T> | undefined): number;
export declare function %[7]s<F, T>(pointer: Pointer<F>, fromSource: (value: F) => T, toSource: (value: T) => F): Pointer<T>;
export declare function %[7]s<F, T>(pointer: Pointer<F> | undefined, fromSource: (value: F) => T, toSource: (value: T) => F): Pointer<T> | undefined;
export declare function %[8]s<T>(identity: object, read: () => T, write: (value: T) => void): Pointer<T>;
export declare function %[9]s(identity: object): import("./types.js").RawPointer;
export declare function %[10]s(left: import("./types.js").RawPointer | undefined, right: import("./types.js").RawPointer | undefined): boolean;
export declare function %[11]s(pointer: import("./types.js").RawPointer | undefined): number;
export interface TsonicAttributeBuilder<T> {
  add(fact: object, ...values: readonly {}[]): void;
  property<TResult>(selector: (value: T) => TResult): TsonicAttributeBuilder<T>;
  method<TResult>(selector: (value: T) => TResult): TsonicAttributeBuilder<T>;
}
export declare function %[12]s<T>(): TsonicAttributeBuilder<T>;
`, addressOf, allocatePointer, loadPointer, storePointer, equalPointer, hashPointer, projectPointer, bindPointer, bindRawPointer, equalRawPointer, hashRawPointer, attribute),
		"lang.js": fmt.Sprintf(`const unsupported = (name) => {
  throw new Error("resolution-only Tsonic core fixture executed " + name);
};
export const %[1]s = () => unsupported("%[1]s");
export const %[2]s = () => unsupported("%[2]s");
export const %[3]s = () => unsupported("%[3]s");
export const %[4]s = () => unsupported("%[4]s");
export const %[5]s = () => unsupported("%[5]s");
export const %[6]s = () => unsupported("%[6]s");
export const %[7]s = () => unsupported("%[7]s");
export const %[8]s = () => unsupported("%[8]s");
export const %[9]s = () => unsupported("%[9]s");
export const %[10]s = () => unsupported("%[10]s");
export const %[11]s = () => unsupported("%[11]s");
const attributeBuilder = Object.freeze({
  add: () => undefined,
  property: () => attributeBuilder,
  method: () => attributeBuilder,
});
export const %[12]s = () => attributeBuilder;
`, addressOf, allocatePointer, loadPointer, storePointer, equalPointer, hashPointer, projectPointer, bindPointer, bindRawPointer, equalRawPointer, hashRawPointer, attribute),
	}, nil
}

func primitiveTypeDeclarations() (string, error) {
	types := []struct {
		symbol     corecontract.Symbol
		underlying string
	}{
		{corecontract.SymbolBool, "boolean"},
		{corecontract.SymbolInt8, "number"},
		{corecontract.SymbolUint8, "number"},
		{corecontract.SymbolInt16, "number"},
		{corecontract.SymbolUint16, "number"},
		{corecontract.SymbolInt32, "number"},
		{corecontract.SymbolUint32, "number"},
		{corecontract.SymbolInt64, "bigint"},
		{corecontract.SymbolUint64, "bigint"},
		{corecontract.SymbolFloat32, "number"},
		{corecontract.SymbolFloat64, "number"},
	}
	var declarations strings.Builder
	for _, primitive := range types {
		name, err := exportName(primitive.symbol)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(
			&declarations,
			"export type %s = %s;\n",
			name,
			primitive.underlying,
		)
	}
	return declarations.String(), nil
}

func exportName(symbol corecontract.Symbol) (string, error) {
	declaration, err := corecontract.Resolve(symbol)
	if err != nil {
		return "", fmt.Errorf("install Tsonic core resolution fixture: %w", err)
	}
	return declaration.Export(), nil
}
