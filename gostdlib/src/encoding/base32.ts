import {
  Encoding,
  hexadecimalEncoding,
  standardEncoding,
} from "../internal/portable/encoding/base32.js";

export { Encoding } from "../internal/portable/encoding/base32.js";

export const state: {
  HexEncoding: Encoding | undefined;
  StdEncoding: Encoding | undefined;
} = {
  HexEncoding: hexadecimalEncoding(),
  StdEncoding: standardEncoding(),
};
