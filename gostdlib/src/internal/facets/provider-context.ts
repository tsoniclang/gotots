import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { bool } from "@gotots/gostdlib/internal/scalars.js";

import type {
  CancelCauseFunc,
  CancelFunc,
  Context as ProviderContext,
} from "../../context.js";
import {
  AfterFunc,
  Cause,
  WithCancel,
  WithCancelCause,
  WithTimeout,
  WithValue,
} from "../../context.js";
import type { Duration } from "../portable/time/duration.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type { Context as ProviderContext } from "../../context.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export function ContextWithValueDirect(
  parent: ProviderContext | undefined,
  key: GoInterfaceValue | undefined,
  value: GoInterfaceValue | undefined,
): ProviderContext {
  return WithValue(parent, key, value);
}

export function ContextWithCancelDirect(
  parent: ProviderContext | undefined,
): [ProviderContext, NonNullable<CancelFunc>] {
  return WithCancel(parent);
}

export function ContextWithCancelCauseDirect(
  parent: ProviderContext | undefined,
): [ProviderContext, NonNullable<CancelCauseFunc>] {
  return WithCancelCause(parent);
}

export function ContextWithTimeoutDirect(
  parent: ProviderContext | undefined,
  timeout: Duration,
): [ProviderContext, NonNullable<CancelFunc>] {
  return WithTimeout(parent, timeout);
}

export function ContextAfterFuncDirect(
  parent: ProviderContext | undefined,
  callback: (() => void) | undefined,
): () => bool {
  return AfterFunc(parent, callback);
}

export function ContextCauseDirect(
  parent: ProviderContext | undefined,
): ProviderErrorInterface | undefined {
  return Cause(parent);
}
