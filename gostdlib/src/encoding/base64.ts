import {
  Encoding,
  standardEncoding,
} from "../internal/portable/encoding/base64.js";

export {
  Encoding,
  NewEncoder,
} from "../internal/portable/encoding/base64.js";

export const state: {
  StdEncoding: Encoding | undefined;
} = {
  StdEncoding: standardEncoding(),
};
