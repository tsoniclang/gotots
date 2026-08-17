import assert from "node:assert/strict";
import test from "node:test";

import { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import type { Type } from "../src/reflect.js";
import {
  createRuntimeType,
  runtimeTypeOf,
  type RuntimeTypeMetadata,
} from "../src/internal/portable/reflect/runtime-type.js";
import {
  pointerDescriptorFor,
  registerRuntimeOpaqueStructValueOperations,
  registerRuntimePointerValueOperations,
  registerRuntimeStructValueOperations,
  registerRuntimeValueOperations,
  type RuntimeValueAdapter,
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

test("typed struct registration owns common guards, locations, and copies", () => {
  type Record = { name: string; count: bigint };
  const stringDescriptor = createRuntimeType(
    () => metadata("string", 24n),
    () => [],
  );
  const intDescriptor = createRuntimeType(
    () => metadata("int64", 2n),
    () => [],
  );
  const recordDescriptor = createRuntimeType(
    () => metadata("example.Record", 25n),
    () => [],
  );
  const recordAdapter = valueAdapter<Record>(Object.freeze({ comparable: true }));
  const stringAdapter = valueAdapter<string>(Object.freeze({ comparable: true }));
  const intAdapter = valueAdapter<bigint>(Object.freeze({ comparable: true }));
  registerRuntimeStructValueOperations(
    recordDescriptor,
    recordAdapter,
    [
      {
        type: () => stringDescriptor,
        settable: true,
        get: (record: Record): GoInterfaceValue => new stringAdapter(record.name),
        set: (record: Record, field: GoInterfaceValue | undefined): void => {
          if (!stringAdapter.$is(field)) {
            throw new Error("foreign string field");
          }
          record.name = field.$go$value;
        },
      },
      {
        type: () => intDescriptor,
        settable: true,
        get: (record: Record): GoInterfaceValue => new intAdapter(record.count),
        set: (record: Record, field: GoInterfaceValue | undefined): void => {
          if (!intAdapter.$is(field)) {
            throw new Error("foreign integer field");
          }
          record.count = field.$go$value;
        },
      },
    ],
    (record: Record): Record => ({ ...record }),
  );

  const source = new recordAdapter({ name: "before", count: 3n });
  const operations = runtimeValueOperations(recordDescriptor);
  assert.equal(operations?.numField, 2n);
  const location = operations?.field?.(source, 0n);
  assert.equal(location?.type(), stringDescriptor);
  assert.equal(location?.settable, true);
  const before = location?.get();
  assert.equal(stringAdapter.$is(before), true);
  if (!stringAdapter.$is(before)) {
    throw new Error("typed string box is absent");
  }
  assert.equal(before.$go$value, "before");
  location?.set(new stringAdapter("after"));
  assert.equal(source.$go$value.name, "after");
  const countLocation = operations?.field?.(source, 1n);
  assert.equal(countLocation?.type(), intDescriptor);
  const count = countLocation?.get();
  assert.equal(intAdapter.$is(count), true);
  if (!intAdapter.$is(count)) {
    throw new Error("typed integer box is absent");
  }
  assert.equal(count.$go$value, 3n);
  countLocation?.set(new intAdapter(5n));
  assert.equal(source.$go$value.count, 5n);
  const cloned = operations?.cloned?.(source);
  assert.equal(recordAdapter.$is(cloned), true);
  if (!recordAdapter.$is(cloned)) {
    throw new Error("typed record clone is absent");
  }
  assert.notEqual(cloned.$go$value, source.$go$value);
  assert.deepEqual(cloned.$go$value, source.$go$value);
  assert.throws(
    () => operations?.field?.(new DynamicValue(Object.freeze({ comparable: true })), 0n),
    panicWith(/foreign interface box/),
  );
  assert.throws(
    () => operations?.field?.(source, 2n),
    panicWith(/index out of range/),
  );
});

test("opaque struct registration preserves exact field obligations", () => {
  type Record = { hidden: string };
  const descriptor = createRuntimeType(
    () => metadata("example.Opaque", 25n),
    () => [],
  );
  const adapter = valueAdapter<Record>(Object.freeze({ comparable: true }));
  registerRuntimeOpaqueStructValueOperations(
    descriptor,
    adapter,
    ["reflect: field Hidden is unavailable"],
  );

  const operations = runtimeValueOperations(descriptor);
  assert.equal(operations?.numField, 1n);
  assert.throws(
    () => operations?.field?.(new adapter({ hidden: "value" }), 0n),
    panicWith(/field Hidden is unavailable/),
  );
  assert.throws(
    () => operations?.field?.(new DynamicValue(Object.freeze({ comparable: true })), 0n),
    panicWith(/foreign interface box/),
  );
});

test("typed pointer registration preserves nil and aliased element storage", () => {
  type Record = { count: bigint };
  type Cell = { value: Record };
  const recordDescriptor = createRuntimeType(
    () => metadata("example.Record", 25n),
    () => [],
  );
  const pointerDescriptor = createRuntimeType(
    () => metadata("*example.Record", 22n),
    () => [],
  );
  const recordAdapter = valueAdapter<Record>(Object.freeze({ comparable: true }));
  const pointerAdapter = valueAdapter<Cell | undefined>(
    Object.freeze({ comparable: true }),
  );
  registerRuntimePointerValueOperations(pointerDescriptor, pointerAdapter, {
    zero: true,
    element: {
      type: () => recordDescriptor,
      get: (cell: Cell): GoInterfaceValue => new recordAdapter(cell.value),
      set: (cell: Cell, value: GoInterfaceValue | undefined): void => {
        if (!recordAdapter.$is(value)) {
          throw new Error("foreign pointee");
        }
        cell.value = value.$go$value;
      },
    },
    newPointer: (): Cell => ({ value: { count: 0n } }),
  });

  const operations = runtimeValueOperations(pointerDescriptor);
  const cell: Cell = { value: { count: 4n } };
  const pointer = new pointerAdapter(cell);
  assert.equal(operations?.isNil?.(pointer), false);
  assert.equal(operations?.isNil?.(new pointerAdapter(undefined)), true);
  const location = operations?.elem?.(pointer);
  assert.equal(location?.type(), recordDescriptor);
  location?.set(new recordAdapter({ count: 9n }));
  assert.equal(cell.value.count, 9n);
  assert.equal(operations?.elem?.(new pointerAdapter(undefined)), undefined);
  const zero = operations?.zero?.();
  assert.equal(pointerAdapter.$is(zero), true);
  if (!pointerAdapter.$is(zero)) {
    throw new Error("typed pointer zero is absent");
  }
  assert.equal(zero.$go$value, undefined);
  const fresh = operations?.newPointer?.();
  assert.equal(pointerAdapter.$is(fresh), true);
  if (!pointerAdapter.$is(fresh) || fresh.$go$value === undefined) {
    throw new Error("typed pointer allocation is absent");
  }
  assert.equal(fresh.$go$value.value.count, 0n);
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

function valueAdapter<T>(
  dynamicType: { readonly comparable: boolean },
): RuntimeValueAdapter<T> {
  return class Adapter extends GoInterfaceValue {
    readonly $go$type = dynamicType;
    readonly $go$methods: ReadonlySet<object> = new Set();
    readonly $go$formatString = false;

    constructor(readonly $go$value: T) {
      super();
    }

    static $is(value: GoInterfaceValue | undefined): value is Adapter {
      return value !== undefined && value.$go$type === dynamicType;
    }

    $go$implements(contract: readonly object[]): boolean {
      return contract.length === 0;
    }

    $go$equal(other: GoInterfaceValue): boolean {
      return Adapter.$is(other) && other.$go$value === this.$go$value;
    }

    $go$hash(): number {
      return 0;
    }

    $go$format(): string {
      return "adapter value";
    }
  };
}

function panicWith(pattern: RegExp): (failure: object) => boolean {
  return (failure: object): boolean => failure instanceof GoPanic
    && pattern.test(failure.value.$go$format("v", "", undefined));
}
