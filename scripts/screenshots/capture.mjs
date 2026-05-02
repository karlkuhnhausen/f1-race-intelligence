// Screenshot capture for blog posts.
//
// Usage:
//   cd scripts/screenshots
//   npm install
//   npx playwright install chromium
//   npm run capture:prod        # against deployed AKS app
//   npm run capture:local       # against local vite dev server (npm run dev in frontend/)
//
// Override page list / viewport / output dir via env vars (see CONFIG below).
//
// Output: docs/blog/images/<page>-<viewport>.png

import { chromium } from 'playwright';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { mkdirSync } from 'node:fs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');

const BASE_URL = process.env.BASE_URL ?? 'https://f1raceintel.westus3.cloudapp.azure.com';
const OUT_DIR = resolve(repoRoot, process.env.OUT_DIR ?? 'docs/blog/images');

// Pages to capture: { name, path, waitFor (CSS selector to wait for before snapping) }
const PAGES = [
  { name: 'calendar',        path: '/',            waitFor: 'table' },
  { name: 'standings',       path: '/standings',   waitFor: 'table' },
  // Round detail — pick a recently completed round so all session types render.
  // Adjust ROUND env var to capture a specific round.
  { name: 'round-detail',    path: `/rounds/${process.env.ROUND ?? '5'}`, waitFor: 'table, .results-table, h1' },
];

const VIEWPORTS = [
  { name: 'desktop', width: 1440, height: 900,  deviceScaleFactor: 2 },
  { name: 'mobile',  width: 390,  height: 844,  deviceScaleFactor: 2 }, // iPhone 14-ish
];

mkdirSync(OUT_DIR, { recursive: true });

const browser = await chromium.launch();
console.log(`Capturing from ${BASE_URL}`);
console.log(`Output → ${OUT_DIR}\n`);

let failures = 0;

for (const vp of VIEWPORTS) {
  const ctx = await browser.newContext({
    viewport: { width: vp.width, height: vp.height },
    deviceScaleFactor: vp.deviceScaleFactor,
    // Deployed AKS uses a self-signed cert on the Azure FQDN; the public
    // browser-trusted cert is on the custom domain. Tolerate either here.
    ignoreHTTPSErrors: true,
  });
  const page = await ctx.newPage();

  for (const p of PAGES) {
    const url = `${BASE_URL}${p.path}`;
    const outFile = resolve(OUT_DIR, `${p.name}-${vp.name}.png`);
    process.stdout.write(`  ${vp.name.padEnd(7)} ${p.name.padEnd(14)} → `);
    try {
      await page.goto(url, { waitUntil: 'networkidle', timeout: 30_000 });
      if (p.waitFor) {
        await page.waitForSelector(p.waitFor, { timeout: 15_000 });
      }
      // Brief settle for fonts/layout shifts
      await page.waitForTimeout(500);
      await page.screenshot({ path: outFile, fullPage: true });
      console.log(`✓ ${outFile.replace(repoRoot + '/', '')}`);
    } catch (err) {
      failures++;
      console.log(`✗ ${err.message}`);
    }
  }
  await ctx.close();
}

await browser.close();

if (failures > 0) {
  console.error(`\n${failures} screenshot(s) failed.`);
  process.exit(1);
}
console.log('\nDone.');
