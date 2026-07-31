import type { GoError } from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int64, uint8 } from "@gotots/runtime/scalars.js";

import type { Time } from "./time.js";

export interface Addr {
  Network(): gostring;
  String(): gostring;
}

export interface Conn {
  Close(): GoError | undefined;
  LocalAddr(): Addr | undefined;
  Read(b: RuntimeSlice<uint8>): [int64, GoError | undefined];
  RemoteAddr(): Addr | undefined;
  SetDeadline(t: Time): GoError | undefined;
  SetReadDeadline(t: Time): GoError | undefined;
  SetWriteDeadline(t: Time): GoError | undefined;
  Write(b: RuntimeSlice<uint8>): [int64, GoError | undefined];
}

export interface Listener {
  Accept(): [Conn | undefined, GoError | undefined];
  Addr(): Addr | undefined;
  Close(): GoError | undefined;
}
