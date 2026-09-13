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
    return ProviderError.fromText("pprof: nil writer");
  }
  if (!beginCpuProfile((content): void => {
    writer.Write(byteSlice(content));
  })) {
    return ProviderError.fromText("cpu profiling already in use");
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
    return ProviderError.fromText("pprof: nil profile or writer");
  }
  const content = profileSnapshot(receiver[ProfileNameKey].text());
  const [count, failure] = writer.Write(byteSlice(content));
  if (failure !== undefined) {
    return failure;
  }
  return count === BigInt(content.length)
    ? undefined
    : ProviderError.fromText("pprof: short write");
}
