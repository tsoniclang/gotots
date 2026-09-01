import { $initialize as $initialize__api } from "./packages/example.com/package-state/api/package.js";
import { $initialize as $initialize__dep } from "./packages/example.com/package-state/dep/package.js";
import { GoCompilationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
type $GoCompilation = never;
attribute<$GoCompilation>().add(GoCompilationFact, "gotots-go-source-compilation-fact-v1", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", "go1.26.4", "linux", "amd64", false, 64, "little-endian", "bigint", "preserve-go", "go-concurrency", "serial-synchronous-execution-envelope", "", "", "", "", 0);
$initialize__dep();
$initialize__api();
