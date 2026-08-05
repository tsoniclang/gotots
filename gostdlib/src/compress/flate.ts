import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

export interface Reader extends GoInterfaceValue {
  Read(buffer: RuntimeSlice<uint8>): [int, GoError | undefined];
  ReadByte(): [uint8, GoError | undefined];
}
