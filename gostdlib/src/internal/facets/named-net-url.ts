import { URL } from "../../net/url.js";

export type NetUrlURLStorage = URL;

export class NetUrlURLOperations {
  static $copy(source: URL): URL {
    const target = new URL();
    NetUrlURLOperations.$assign(target, source);
    return target;
  }

  static $assign(target: URL, source: URL): void {
    const snapshot = {
      Scheme: source.Scheme,
      Opaque: source.Opaque,
      User: source.User,
      Host: source.Host,
      Path: source.Path,
      Fragment: source.Fragment,
      RawQuery: source.RawQuery,
      RawPath: source.RawPath,
      RawFragment: source.RawFragment,
      ForceQuery: source.ForceQuery,
      OmitHost: source.OmitHost,
    };
    target.Scheme = snapshot.Scheme;
    target.Opaque = snapshot.Opaque;
    target.User = snapshot.User;
    target.Host = snapshot.Host;
    target.Path = snapshot.Path;
    target.Fragment = snapshot.Fragment;
    target.RawQuery = snapshot.RawQuery;
    target.RawPath = snapshot.RawPath;
    target.RawFragment = snapshot.RawFragment;
    target.ForceQuery = snapshot.ForceQuery;
    target.OmitHost = snapshot.OmitHost;
  }

  static $storageOf(source: URL): NetUrlURLStorage {
    return source;
  }

  static $fromStorage(source: NetUrlURLStorage): URL {
    return source;
  }
}
