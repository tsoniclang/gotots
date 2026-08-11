import assert from "node:assert/strict";
import test from "node:test";

import { GoInterfaceValue } from "@gotots/runtime/interface-value.js";

import type { Type } from "../src/reflect.js";
import {
  createRuntimeType,
  runtimeTypeOf,
  type RuntimeTypeMetadata,
} from "../src/internal/portable/reflect/runtime-type.js";
import {
  pointerDescriptorFor,
  registerRuntimeValueOperations,
  runtimeValueOperations,
} from "../src/internal/portable/reflect/runtime-value.js";

class DynamicValue extends GoInterfaceValue {
  readonly $go$methods: ReadonlySet<object> = new Set();
  readonly $go$formatString = false;

  constructor(readonly $go$type: { readonly comparable: boolean }) {
    super();
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.length === 0;
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  $go$format(): string {
    return "dynamic value";
  }
}

test("reflection metadata and value operations materialize exactly on demand", () => {
  const dynamicType = Object.freeze({ comparable: true });
  const methodToken = Object.freeze({});
  let metadataCalls = 0;
  let methodCalls = 0;
  let operationCalls = 0;
  const descriptor = createRuntimeType(
    (): RuntimeTypeMetadata => {
      metadataCalls += 1;
      return metadata("example.Value", 25n);
    },
    (): readonly object[] => {
      methodCalls += 1;
      return [methodToken];
    },
    { dynamicType },
  );
  registerRuntimeValueOperations(descriptor, () => {
    operationCalls += 1;
    return { isZero: () => true };
  });

  assert.deepEqual([metadataCalls, methodCalls, operationCalls], [0, 0, 0]);
  assert.equal(runtimeTypeOf(new DynamicValue(dynamicType)), descriptor);
  assert.deepEqual([metadataCalls, methodCalls, operationCalls], [0, 0, 0]);

  assert.equal(descriptor.String(), "example.Value");
  assert.equal(descriptor.String(), "example.Value");
  assert.equal(descriptor.$go$methods.has(methodToken), true);
  assert.equal(descriptor.$go$methods.has(methodToken), true);
  assert.equal(runtimeValueOperations(descriptor)?.isZero?.(new DynamicValue(dynamicType)), true);
  assert.equal(runtimeValueOperations(descriptor)?.isZero?.(new DynamicValue(dynamicType)), true);
  assert.deepEqual([metadataCalls, methodCalls, operationCalls], [1, 1, 1]);
});

test("lazy pointer registration preserves the predeclared descriptor identity", () => {
  const element = createRuntimeType(
    () => metadata("int", 2n),
    () => [],
  );
  let pointerMetadataCalls = 0;
  let elementCalls = 0;
  const pointer = createRuntimeType(
    () => {
      pointerMetadataCalls += 1;
      return metadata("*int", 22n);
    },
    () => [],
    {
      pointerElement: () => {
        elementCalls += 1;
        return element;
      },
    },
  );

  assert.deepEqual([pointerMetadataCalls, elementCalls], [0, 0]);
  assert.equal(pointerDescriptorFor(element), pointer);
  assert.deepEqual([pointerMetadataCalls, elementCalls], [0, 1]);
  assert.equal(pointer.Elem(), element);
  assert.deepEqual([pointerMetadataCalls, elementCalls], [0, 2]);
});

function metadata(text: string, kind: bigint): RuntimeTypeMetadata {
  return {
    identity: text,
    kind,
    text,
    size: 8n,
    align: 8n,
  };
}
