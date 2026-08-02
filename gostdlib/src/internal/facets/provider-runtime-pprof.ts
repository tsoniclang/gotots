import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  beginCpuProfile,
  type ProfileIdentity,
  ProfileNameKey,
  profileSnapshot,
} from "../node/runtime/profile.js";
import { byteSlice } from "../runtime/slice.js";
import {
  CanonicalBoundaryErrorAsync,
  CanonicalBoundaryErrorSync,
} from "./provider-io-contract.js";
import type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

type ErrorFactory<Failure extends GoInterfaceValue> = (
  message: string,
) => Failure;

export function PprofStartCPUProfileCanonicalSyncWriterSyncError(
  writer: CanonicalWriterTargetSync<CanonicalErrorSync> | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return startWithSyncWriter(
    writer,
    (message) => new CanonicalBoundaryErrorSync(message, errorContract),
  );
}

export function PprofStartCPUProfileCanonicalAsyncWriterSyncError(
  writer: CanonicalWriterTargetAsync<CanonicalErrorSync> | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return startWithAsyncWriter(
    writer,
    (message) => new CanonicalBoundaryErrorSync(message, errorContract),
  );
}

export function PprofStartCPUProfileCanonicalSyncWriterAsyncError(
  writer: CanonicalWriterTargetSync<CanonicalErrorAsync> | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return startWithSyncWriter(
    writer,
    (message) => new CanonicalBoundaryErrorAsync(message, errorContract),
  );
}

export function PprofStartCPUProfileCanonicalAsyncWriterAsyncError(
  writer: CanonicalWriterTargetAsync<CanonicalErrorAsync> | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return startWithAsyncWriter(
    writer,
    (message) => new CanonicalBoundaryErrorAsync(message, errorContract),
  );
}

export function PprofProfileWriteToCanonicalSyncWriterSyncError(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriterTargetSync<CanonicalErrorSync> | undefined,
  debug: int64,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return writeProfileSync(
    receiver,
    writer,
    debug,
    (message) => new CanonicalBoundaryErrorSync(message, errorContract),
  );
}

export function PprofProfileWriteToCanonicalAsyncWriterSyncError(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriterTargetAsync<CanonicalErrorSync> | undefined,
  debug: int64,
  errorContract: readonly object[],
): Promise<CanonicalErrorSync | undefined> {
  return writeProfileAsync(
    receiver,
    writer,
    debug,
    (message) => new CanonicalBoundaryErrorSync(message, errorContract),
  );
}

export function PprofProfileWriteToCanonicalSyncWriterAsyncError(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriterTargetSync<CanonicalErrorAsync> | undefined,
  debug: int64,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return writeProfileSync(
    receiver,
    writer,
    debug,
    (message) => new CanonicalBoundaryErrorAsync(message, errorContract),
  );
}

export function PprofProfileWriteToCanonicalAsyncWriterAsyncError(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriterTargetAsync<CanonicalErrorAsync> | undefined,
  debug: int64,
  errorContract: readonly object[],
): Promise<CanonicalErrorAsync | undefined> {
  return writeProfileAsync(
    receiver,
    writer,
    debug,
    (message) => new CanonicalBoundaryErrorAsync(message, errorContract),
  );
}

function startWithSyncWriter<Failure extends GoInterfaceValue>(
  writer: CanonicalWriterTargetSync<Failure> | undefined,
  createError: ErrorFactory<Failure>,
): Failure | undefined {
  if (writer === undefined) {
    return createError("pprof: nil writer");
  }
  if (!beginCpuProfile(async (content): Promise<void> => {
    writer.Write(byteSlice(content));
  })) {
    return createError("cpu profiling already in use");
  }
  return undefined;
}

function startWithAsyncWriter<Failure extends GoInterfaceValue>(
  writer: CanonicalWriterTargetAsync<Failure> | undefined,
  createError: ErrorFactory<Failure>,
): Failure | undefined {
  if (writer === undefined) {
    return createError("pprof: nil writer");
  }
  if (!beginCpuProfile(async (content): Promise<void> => {
    await writer.Write(byteSlice(content));
  })) {
    return createError("cpu profiling already in use");
  }
  return undefined;
}

function writeProfileSync<Failure extends GoInterfaceValue>(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriterTargetSync<Failure> | undefined,
  debug: int64,
  createError: ErrorFactory<Failure>,
): Failure | undefined {
  void debug;
  if (receiver === undefined || writer === undefined) {
    return createError("pprof: nil profile or writer");
  }
  const content = profileSnapshot(receiver[ProfileNameKey]);
  const [count, failure] = writer.Write(byteSlice(content));
  return writeResult(count, content, failure, createError);
}

async function writeProfileAsync<Failure extends GoInterfaceValue>(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriterTargetAsync<Failure> | undefined,
  debug: int64,
  createError: ErrorFactory<Failure>,
): Promise<Failure | undefined> {
  void debug;
  if (receiver === undefined || writer === undefined) {
    return createError("pprof: nil profile or writer");
  }
  const content = profileSnapshot(receiver[ProfileNameKey]);
  const [count, failure] = await writer.Write(byteSlice(content));
  return writeResult(count, content, failure, createError);
}

function writeResult<Failure extends GoInterfaceValue>(
  count: int64,
  content: Uint8Array,
  failure: Failure | undefined,
  createError: ErrorFactory<Failure>,
): Failure | undefined {
  if (failure !== undefined) {
    return failure;
  }
  return count === content.length
    ? undefined
    : createError("pprof: short write");
}
