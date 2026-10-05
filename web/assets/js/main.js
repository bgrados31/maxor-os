// Maxor OS website. Plain ES modules, no build step and no third-party code.
import { initI18n } from "./i18n.js";
import { initEffects } from "./effects.js";
import { initDesk } from "./desk.js";
import { initRelease } from "./release.js";

await initI18n();
initEffects();
initRelease();
initDesk();
