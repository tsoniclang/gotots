import { gunzipSync } from "node:zlib";

export function decodeGzip(source: Uint8Array): Uint8Array {
  return Uint8Array.from(gunzipSync(source));
}
