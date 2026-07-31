import { URL } from "../../net/url.js";

export type NetUrlURLStorage = URL;

export class NetUrlURLOperations {
  static $storageOf(source: URL): NetUrlURLStorage {
    return source;
  }

  static $fromStorage(source: NetUrlURLStorage): URL {
    return source;
  }
}
