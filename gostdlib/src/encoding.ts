import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { uint8 } from "@gotots/gostdlib/internal/scalars.js";

export interface TextMarshaler extends GoInterfaceValue {
  MarshalText(): [RuntimeSlice<uint8>, GoError | undefined];
}
