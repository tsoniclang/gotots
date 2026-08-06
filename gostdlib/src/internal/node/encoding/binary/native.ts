import { endianness } from "node:os";

import {
  BigEndianOrder,
  EndianOrder,
  LittleEndianOrder,
} from "../../../portable/encoding/binary/byte-order.js";

export function nativeEndian(): EndianOrder {
  return endianness() === "BE" ? new BigEndianOrder() : new LittleEndianOrder();
}
