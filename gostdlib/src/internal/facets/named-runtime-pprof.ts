import { Profile } from "../../runtime/pprof.js";
import { ProfileNameKey } from "../node/runtime/profile.js";

export class RuntimePprofProfileOperations {
  static $copy(source: Profile): Profile {
    return new Profile(source[ProfileNameKey]);
  }

  static $assign(target: Profile, source: Profile): void {
    target[ProfileNameKey] = source[ProfileNameKey];
  }
}
