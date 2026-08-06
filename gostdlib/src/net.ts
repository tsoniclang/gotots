import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import type { Time } from "./time.js";
import { ProviderError } from "./internal/runtime/error.js";

export interface Addr extends GoInterfaceValue {
  Network(): gostring;
  String(): gostring;
}

export interface Conn extends GoInterfaceValue {
  Close(): GoError | undefined;
  LocalAddr(): Addr | undefined;
  Read(b: RuntimeSlice<uint8>): [int, GoError | undefined];
  RemoteAddr(): Addr | undefined;
  SetDeadline(t: Time): GoError | undefined;
  SetReadDeadline(t: Time): GoError | undefined;
  SetWriteDeadline(t: Time): GoError | undefined;
  Write(b: RuntimeSlice<uint8>): [int, GoError | undefined];
}

export interface Listener extends GoInterfaceValue {
  Accept(): [Conn | undefined, GoError | undefined];
  Addr(): Addr | undefined;
  Close(): GoError | undefined;
}

export function Listen(
  network: gostring,
  _address: gostring,
): [Listener | undefined, GoError | undefined] {
  return [
    undefined,
    new ProviderError(`net.Listen ${network} is not implemented by this backend`),
  ];
}
