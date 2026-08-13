import { $initialize as $initialize__api } from "./packages/example.com/package-initialization/api/package.js";
import { $initialize as $initialize__sideeffect } from "./packages/example.com/package-initialization/sideeffect/package.js";
import { $initialize as $initialize__sink } from "./packages/example.com/package-initialization/sink/package.js";
$initialize__sink();
$initialize__sideeffect();
$initialize__api();
