import {
  closeSync,
  readSync,
  writeSync,
} from "node:fs";
import type { GoError } from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64, uint64, uint8 } from "@gotots/runtime/scalars.js";
import { state as ioState } from "../../../io.js";
import { state as fsState } from "../../../io/fs.js";
import {
  bytes,
  writeBytes,
} from "../../runtime/slice.js";
import { nodeError } from "./error.js";

const invalidDescriptor = 0xffffffffffffffff;

interface FileDescriptor {
  readonly descriptor: number;
  readonly name: string;
  readonly closeDescriptor: boolean;
  closed: boolean;
}

const descriptors = new WeakMap<object, FileDescriptor>();

export function attachFileDescriptor(
  file: object,
  descriptor: number,
  name: string,
): void {
  descriptors.set(file, {
    descriptor,
    name,
    closeDescriptor: true,
    closed: false,
  });
}

export function attachStandardFileDescriptor(
  file: object,
  descriptor: number,
  name: string,
): void {
  descriptors.set(file, {
    descriptor,
    name,
    closeDescriptor: true,
    closed: false,
  });
}

export function closeFile(receiver: object | undefined): GoError | undefined {
  if (receiver === undefined) {
    return fsState.ErrInvalid;
  }
  const state = descriptorOf(receiver);
  if (state === undefined) {
    return nodeError("invalid", "close");
  }
  if (state.closed) {
    return nodeError("closed", "close", state.name);
  }
  if (state.closeDescriptor) {
    try {
      closeSync(state.descriptor);
    } catch {
      return nodeError("operation", "close", state.name);
    }
  }
  state.closed = true;
  return undefined;
}

export function fileDescriptor(receiver: object | undefined): uint64 {
  if (receiver === undefined) {
    return invalidDescriptor;
  }
  const state = descriptorOf(receiver);
  if (state === undefined || state.closed) {
    return invalidDescriptor;
  }
  return state.descriptor;
}

export function readFile(
  receiver: object | undefined,
  buffer: RuntimeSlice<uint8>,
): [int64, GoError | undefined] {
  if (receiver === undefined) {
    return [0, fsState.ErrInvalid];
  }
  const state = descriptorOf(receiver);
  if (state === undefined) {
    return [0, nodeError("invalid", "read")];
  }
  if (state.closed) {
    return [0, nodeError("closed", "read", state.name)];
  }
  const target = new Uint8Array(buffer.length);
  try {
    const count = readSync(
      state.descriptor,
      target,
      0,
      target.length,
      null,
    );
    if (count === 0) {
      return [0, ioState.EOF];
    }
    writeBytes(buffer, target.subarray(0, count));
    return [count, undefined];
  } catch {
    return [0, nodeError("operation", "read", state.name)];
  }
}

export function writeFile(
  receiver: object | undefined,
  buffer: RuntimeSlice<uint8>,
): [int64, GoError | undefined] {
  if (receiver === undefined) {
    return [0, fsState.ErrInvalid];
  }
  const state = descriptorOf(receiver);
  if (state === undefined) {
    return [0, nodeError("invalid", "write")];
  }
  if (state.closed) {
    return [0, nodeError("closed", "write", state.name)];
  }
  try {
    const count = writeSync(
      state.descriptor,
      bytes(buffer),
      0,
      buffer.length,
      null,
    );
    return [count, undefined];
  } catch {
    return [0, nodeError("operation", "write", state.name)];
  }
}

export function writeFileString(
  receiver: object | undefined,
  text: gostring,
): [int64, GoError | undefined] {
  if (receiver === undefined) {
    return [0, fsState.ErrInvalid];
  }
  const state = descriptorOf(receiver);
  if (state === undefined) {
    return [0, nodeError("invalid", "write")];
  }
  if (state.closed) {
    return [0, nodeError("closed", "write", state.name)];
  }
  try {
    return [writeSync(state.descriptor, text), undefined];
  } catch {
    return [0, nodeError("operation", "write", state.name)];
  }
}

function descriptorOf(file: object): FileDescriptor | undefined {
  return descriptors.get(file);
}
