// Capture analysis-specific screenshots for blog post.
//
// Usage:
//   cd scripts/screenshots
//   node capture-analysis.mjs
//
// Output: docs/blog/images/analysis-*.png

import { chromium } from 'playwright';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { mkdirSync } from 'node:fs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');

const BASE_URL = process.env.BASE_URL ?? 'http://f1raceintel.westus3.cloudapp.azure.com';
const OUT_DIR = resolve(repoRoot, 'docs/blog/images');
const ROUND = process.env.ROUND ?? '1';

mkdirSync(OUT_DIR, { recursive: true });

const browser = await chromium.launch();
const ctx = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  deviceScaleFactor: 2,
});

const page = await ctx.newPage();

// 1. Screenshot: "View Analysis" button on round detail page
console.log(`Navigating to round ${ROUND} detail page...`);
await page.goto(`${BASE_URL}/rounds/${ROUND}?year=2026`, { waitUntil: 'networkidle' });
await page.waitForTimeout(2000);

// Find the View Analysis link and scroll to it, then capture surrounding area
const analysisLink = page.locator('a:has-text("View Analysis")').first();
if (await analysisLink.isVisible()) {
  // Scroll the link into view and capture a wider area around it
  await analysisLink.scrollIntoViewIfNeeded();
  await page.waitForTimeout(500);
  // Take a screenshot of the section containing the button
  const box = await analysisLink.boundingBox();
  if (box) {
    // Capture a region around the button showing context
    const clip = {
      x: Math.max(0, box.x - 40),
      y: Math.max(0, box.y - 200),
      width: Math.min(1440, box.width + 600),
      height: 300,
    };
    await page.screenshot({
      path: resolve(OUT_DIR, 'analysis-view-button.png'),
      clip,
    });
    console.log('  ✓ analysis-view-button.png');
  }
} else {
  console.log('  ✗ View Analysis button not found');
}

// 2. Navigate to the analysis page
console.log(`Navigating to race analysis page...`);
await page.goto(`${BASE_URL}/rounds/${ROUND}/sessions/race/analysis`, { waitUntil: 'networkidle' });
await page.waitForTimeout(3000);

// Helper: screenshot a section by heading text
async function captureSection(headingText, filename) {
  const heading = page.locator(`h3:has-text("${headingText}")`);
  if (await heading.isVisible()) {
    // Get the parent <section> element
    const section = heading.locator('..');
    await section.scrollIntoViewIfNeeded();
    await page.waitForTimeout(500);
    await section.screenshot({ path: resolve(OUT_DIR, filename) });
    console.log(`  ✓ ${filename}`);
  } else {
    console.log(`  ✗ ${headingText} section not found`);
  }
}

await captureSection('Position Battle', 'analysis-position-battle.png');
await captureSection('Gap to Leader', 'analysis-gap-to-leader.png');
await captureSection('Tire Strategy', 'analysis-tire-strategy.png');
await captureSection('Pit Stops', 'analysis-pit-stops.png');

// 3. Take a full-page screenshot too for reference
await page.evaluate(() => window.scrollTo(0, 0));
await page.waitForTimeout(300);
await page.screenshot({
  path: resolve(OUT_DIR, 'analysis-full-page.png'),
  fullPage: true,
});
console.log('  ✓ analysis-full-page.png');

await browser.close();
console.log('\nDone.');
