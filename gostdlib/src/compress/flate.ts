import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

export interface Reader extends GoInterfaceValue {
  Read(buffer: RuntimeSlice<uint8>): [int64, GoError | undefined];
  ReadByte(): [uint8, GoError | undefined];
}
