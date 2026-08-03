import type { int64 } from "@gotots/runtime/scalars.js";

import {
  beginCpuProfile,
  type ProfileIdentity,
  ProfileNameKey,
  profileSnapshot,
} from "../node/runtime/profile.js";
import { byteSlice } from "../runtime/slice.js";
import {
  CanonicalBoundaryError,
  type CanonicalError,
  type CanonicalWriter,
} from "./provider-io-contract.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalWriter,
} from "./provider-io-contract.js";

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
  debug: int64,
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
  count: int64,
  content: Uint8Array,
  failure: CanonicalError | undefined,
  errorContract: InterfaceContract,
): CanonicalError | undefined {
  if (failure !== undefined) {
    return failure;
  }
  return count === content.length
    ? undefined
    : new CanonicalBoundaryError("pprof: short write", errorContract);
}
