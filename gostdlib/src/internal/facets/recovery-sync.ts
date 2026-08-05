import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/gostdlib/internal/scalars.js";

import {
  Mutex,
  Pool,
  RWMutex,
  WaitGroup,
} from "../../sync.js";
import { Bool } from "../../sync/atomic.js";

export function SyncMutexUnlock(
  receiver: Mutex | undefined,
  _recovery?: GoRecovery,
): void {
  Mutex.Unlock(receiver);
}

export function SyncPoolPut(
  receiver: Pool | undefined,
  value: GoInterfaceValue | undefined,
  _recovery?: GoRecovery,
): void {
  Pool.Put(receiver, value);
}

export function SyncRWMutexRUnlock(
  receiver: RWMutex | undefined,
  _recovery?: GoRecovery,
): void {
  RWMutex.RUnlock(receiver);
}

export function SyncRWMutexUnlock(
  receiver: RWMutex | undefined,
  _recovery?: GoRecovery,
): void {
  RWMutex.Unlock(receiver);
}

export function SyncWaitGroupDone(
  receiver: WaitGroup | undefined,
  _recovery?: GoRecovery,
): void {
  WaitGroup.Done(receiver);
}

export function SyncAtomicBoolStore(
  receiver: Bool | undefined,
  value: bool,
  _recovery?: GoRecovery,
): void {
  Bool.Store(receiver, value);
}
