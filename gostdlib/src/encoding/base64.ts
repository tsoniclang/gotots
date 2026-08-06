import {
  Encoding,
  standardEncoding,
  urlEncoding,
} from "../internal/portable/encoding/base64.js";

export {
  Encoding,
  NewEncoder,
} from "../internal/portable/encoding/base64.js";

export const state: {
  StdEncoding: Encoding | undefined;
  URLEncoding: Encoding | undefined;
} = {
  StdEncoding: standardEncoding(),
  URLEncoding: urlEncoding(),
};
