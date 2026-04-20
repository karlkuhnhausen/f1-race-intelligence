// @vitest-environment node
import { describe, it, expect } from "vitest";
import * as fs from "node:fs";
import * as path from "node:path";

/**
 * Network-boundary compliance test.
 *
 * The frontend must NEVER call OpenF1 or Hyprace APIs directly.
 * All data flows through the backend /api/v1/* endpoints.
 * This test scans every .ts/.tsx source file to catch violations.
 */

const FORBIDDEN_PATTERNS = [
  /api\.openf1\.org/i,
  /openf1\.org/i,
  /hyprace\.io/i,
  /hyprace\.com/i,
];

function collectSourceFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory() && entry.name !== "node_modules" && entry.name !== "dist") {
      results.push(...collectSourceFiles(full));
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      results.push(full);
    }
  }
  return results;
}

describe("Network boundary: no direct external API calls", () => {
  const srcDir = path.resolve(__dirname, "../../src");
  const files = collectSourceFiles(srcDir);

  it("should find source files to scan", () => {
    expect(files.length).toBeGreaterThan(0);
  });

  it("should not contain any reference to OpenF1 or Hyprace domains", () => {
    const violations: string[] = [];

    for (const filePath of files) {
      const content = fs.readFileSync(filePath, "utf-8");
      const lines = content.split("\n");

      for (let i = 0; i < lines.length; i++) {
        for (const pattern of FORBIDDEN_PATTERNS) {
          if (pattern.test(lines[i])) {
            const rel = path.relative(srcDir, filePath);
            violations.push(`${rel}:${i + 1} matches ${pattern}`);
          }
        }
      }
    }

    expect(violations).toEqual([]);
  });

  it("should use apiClient for all data fetching", () => {
    const apiFiles = files.filter(
      (f) => f.endsWith("Api.ts") || f.endsWith("api.ts")
    );

    for (const filePath of apiFiles) {
      const content = fs.readFileSync(filePath, "utf-8");
      expect(content).toContain("apiClient");
    }
  });
});
