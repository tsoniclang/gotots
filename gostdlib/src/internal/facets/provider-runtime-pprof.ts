import type { int } from "@gotots/gostdlib/internal/scalars.js";

import {
  beginCpuProfile,
  type ProfileIdentity,
  ProfileNameKey,
  profileSnapshot,
} from "../node/runtime/profile.js";
import { byteSlice } from "../runtime/slice.js";
import { ProviderError } from "../runtime/error.js";
import {
  CanonicalBoundaryError,
  type CanonicalError,
  type CanonicalWriter,
  type ProviderWriterInterface,
} from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalWriter,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export function PprofStartCPUProfileDirect(
  writer: ProviderWriterInterface<ProviderErrorInterface> | undefined,
): ProviderErrorInterface | undefined {
  if (writer === undefined) {
    return new ProviderError("pprof: nil writer");
  }
  if (!beginCpuProfile((content): void => {
    writer.Write(byteSlice(content));
  })) {
    return new ProviderError("cpu profiling already in use");
  }
  return undefined;
}

export function PprofProfileWriteToDirect(
  receiver: ProfileIdentity | undefined,
  writer: ProviderWriterInterface<ProviderErrorInterface> | undefined,
  debug: int,
): ProviderErrorInterface | undefined {
  void debug;
  if (receiver === undefined || writer === undefined) {
    return new ProviderError("pprof: nil profile or writer");
  }
  const content = profileSnapshot(receiver[ProfileNameKey]);
  const [count, failure] = writer.Write(byteSlice(content));
  if (failure !== undefined) {
    return failure;
  }
  return count === BigInt(content.length)
    ? undefined
    : new ProviderError("pprof: short write");
}

export function PprofStartCPUProfileCanonical(
  writer: CanonicalWriter<CanonicalError> | undefined,
  errorContract: InterfaceContract,
): CanonicalError | undefined {
  if (writer === undefined) {
    return new CanonicalBoundaryError("pprof: nil writer", errorContract);
  }
  if (!beginCpuProfile(async (content): Promise<void> => {
    await writer.Write(byteSlice(content));
  })) {
    return new CanonicalBoundaryError(
      "cpu profiling already in use",
      errorContract,
    );
  }
  return undefined;
}

export async function PprofProfileWriteToCanonical(
  receiver: ProfileIdentity | undefined,
  writer: CanonicalWriter<CanonicalError> | undefined,
  debug: int,
  errorContract: InterfaceContract,
): Promise<CanonicalError | undefined> {
  void debug;
  if (receiver === undefined || writer === undefined) {
    return new CanonicalBoundaryError(
      "pprof: nil profile or writer",
      errorContract,
    );
  }
  const content = profileSnapshot(receiver[ProfileNameKey]);
  const [count, failure] = await writer.Write(byteSlice(content));
  return writeResult(count, content, failure, errorContract);
}

function writeResult(
  count: int,
  content: Uint8Array,
  failure: CanonicalError | undefined,
  errorContract: InterfaceContract,
): CanonicalError | undefined {
  if (failure !== undefined) {
    return failure;
  }
  return count === BigInt(content.length)
    ? undefined
    : new CanonicalBoundaryError("pprof: short write", errorContract);
}
