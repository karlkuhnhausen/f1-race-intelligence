// Capture screenshots for Day 24 and Day 25 blog posts.
//
// Day 24: Circuit-name X-axis labels on standings progression chart
// Day 25: Miami sprint session results fix (all 5 sessions with data)
//
// Usage:
//   cd scripts/screenshots
//   node capture-day24-25.mjs

import { chromium } from 'playwright';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { mkdirSync } from 'node:fs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');

const BASE_URL = process.env.BASE_URL ?? 'http://f1raceintel.westus3.cloudapp.azure.com';
const OUT_DIR = resolve(repoRoot, 'docs/blog/images');

mkdirSync(OUT_DIR, { recursive: true });

const browser = await chromium.launch();
const ctx = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 2,
  ignoreHTTPSErrors: true,
});

const page = await ctx.newPage();

// ── Day 24: Standings progression chart with circuit-name labels ──

console.log('Day 24: Standings progression chart...');
await page.goto(`${BASE_URL}/standings?year=2026`, { waitUntil: 'networkidle', timeout: 30_000 });
await page.waitForTimeout(2000);

// Click the "Chart" or progression tab to show the chart view
const chartTab = page.locator('button:has-text("Chart"), [role="tab"]:has-text("Chart"), button:has-text("Progression")');
if (await chartTab.count() > 0) {
  await chartTab.first().click();
  await page.waitForTimeout(2000);
}

// Try to capture the driver progression chart section
await page.screenshot({
  path: resolve(OUT_DIR, 'day-24-standings-progression.png'),
  fullPage: true,
});
console.log('  ✓ day-24-standings-progression.png');

// Also capture just the chart area if possible
const chartContainer = page.locator('.recharts-responsive-container, .recharts-wrapper, [class*="chart"]').first();
if (await chartContainer.isVisible().catch(() => false)) {
  await chartContainer.scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  await chartContainer.screenshot({
    path: resolve(OUT_DIR, 'day-24-chart-closeup.png'),
  });
  console.log('  ✓ day-24-chart-closeup.png');
}

// ── Day 25: Miami round detail — all 5 sessions with results ──

console.log('\nDay 25: Miami round detail (round 4)...');
await page.goto(`${BASE_URL}/rounds/4?year=2026`, { waitUntil: 'networkidle', timeout: 30_000 });
await page.waitForTimeout(3000);

// Full page screenshot showing all sessions
await page.screenshot({
  path: resolve(OUT_DIR, 'day-25-miami-full.png'),
  fullPage: true,
});
console.log('  ✓ day-25-miami-full.png');

// Capture the session headers/tabs area showing all 5 session types
const sessionHeaders = page.locator('h1, h2, [class*="session"]').first();
if (await sessionHeaders.isVisible().catch(() => false)) {
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.waitForTimeout(300);
  await page.screenshot({
    path: resolve(OUT_DIR, 'day-25-miami-header.png'),
    clip: { x: 0, y: 0, width: 1440, height: 600 },
  });
  console.log('  ✓ day-25-miami-header.png');
}

// Try to find and screenshot a sprint results section
const sprintSection = page.locator('text=Sprint').first();
if (await sprintSection.isVisible().catch(() => false)) {
  await sprintSection.scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  // Capture area around the sprint section
  const box = await sprintSection.boundingBox();
  if (box) {
    const clip = {
      x: 0,
      y: Math.max(0, box.y - 50),
      width: 1440,
      height: 600,
    };
    await page.screenshot({
      path: resolve(OUT_DIR, 'day-25-miami-sprint.png'),
      clip,
    });
    console.log('  ✓ day-25-miami-sprint.png');
  }
}

// Also grab the sprint qualifying section
const sqSection = page.locator('text=Sprint Qualifying').first();
if (await sqSection.isVisible().catch(() => false)) {
  await sqSection.scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  const box = await sqSection.boundingBox();
  if (box) {
    const clip = {
      x: 0,
      y: Math.max(0, box.y - 50),
      width: 1440,
      height: 600,
    };
    await page.screenshot({
      path: resolve(OUT_DIR, 'day-25-miami-sprint-qualifying.png'),
      clip,
    });
    console.log('  ✓ day-25-miami-sprint-qualifying.png');
  }
}

// Capture a non-sprint round for comparison (Round 3 = Japan, standard weekend)
console.log('\nDay 25: Japan round detail (round 3, standard weekend)...');
await page.goto(`${BASE_URL}/rounds/3?year=2026`, { waitUntil: 'networkidle', timeout: 30_000 });
await page.waitForTimeout(2000);

await page.screenshot({
  path: resolve(OUT_DIR, 'day-25-japan-full.png'),
  fullPage: true,
});
console.log('  ✓ day-25-japan-full.png');

await browser.close();
console.log('\nDone.');
