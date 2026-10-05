// Maxor OS website. Plain ES modules, no build step; the only libraries are vendored (assets/vendor).
import { initI18n } from "./i18n.js";
import { fillMarquee, initMotion } from "./motion.js";
import { initEffects } from "./effects.js";
import { initDesk } from "./desk.js";
import { initRelease } from "./release.js";
import { initTerm } from "./term.js";

await initI18n();
if (!initMotion()) fillMarquee();
initEffects();
initRelease();
initDesk();
initTerm();
