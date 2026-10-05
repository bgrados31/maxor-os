// Renders the raster brand assets from their sources, with the site's own fonts and CSS:
//   assets/brand/og.png (social card, from tools/og.html) and assets/brand/apple-touch-icon.png (from tools/touch-icon.html).
// usage (from web/): node tools/render-images.mjs   — needs Playwright and a Chromium (CHROMIUM=/path/to/chrome).
import { fileURLToPath } from "node:url";

// PLAYWRIGHT=/path/to/playwright/index.mjs when it is installed globally
const { chromium } = await import(process.env.PLAYWRIGHT ?? "playwright");
const web = fileURLToPath(new URL("..", import.meta.url));
const browser = await chromium.launch(process.env.CHROMIUM ? { executablePath: process.env.CHROMIUM } : {});
const page = await browser.newPage({ viewport: { width: 1200, height: 630 }, reducedMotion: "reduce" });
await page.goto(`file://${web}tools/og.html`);
await page.evaluate(() => document.fonts.ready);
await page.screenshot({ path: `${web}assets/brand/og.png` });
await page.setViewportSize({ width: 180, height: 180 });
await page.goto(`file://${web}tools/touch-icon.html`);
await page.screenshot({ path: `${web}assets/brand/apple-touch-icon.png` });
await browser.close();
