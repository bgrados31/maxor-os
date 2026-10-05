# Vendored libraries

Served from the site itself (the CSP allows no other origin). Update by replacing the files from npm.

| File | Package | Licence |
|---|---|---|
| `gsap.min.js`, `ScrollTrigger.min.js` | gsap 3.15.0 | GSAP Standard "no charge" licence, https://gsap.com/standard-license |
| `lenis.min.js` | lenis 1.3.26 | MIT (`LICENSE-lenis.txt`) |

The site works without them: motion.js checks for `window.gsap` and the page falls back to its CSS reveals.
