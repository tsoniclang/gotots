import type { int } from "@gotots/gostdlib/internal/scalars.js";

import {
  beginCpuProfile,
  type ProfileIdentity,
  ProfileNameKey,
  profileSnapshot,
} from "../node/runtime/profile.js";
import { ProviderError } from "../runtime/error.js";
import { byteSlice } from "../runtime/slice.js";
import type { ProviderWriterInterface } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type { ProviderWriterInterface } from "./provider-io-contract.js";
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
