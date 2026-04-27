/**
 * Team color mapping for F1 constructors.
 * Returns hex color used for accent borders, badges, and row stripes.
 *
 * Lookup is case-insensitive and handles both canonical OpenF1 IDs
 * (e.g., "red_bull") and display names from API responses
 * (e.g., "Red Bull Racing").
 */

export const TEAM_COLORS: Record<string, string> = {
  // Mercedes
  mercedes: "#00d2be",
  // Red Bull
  redbull: "#3671c6",
  red_bull: "#3671c6",
  red_bull_racing: "#3671c6",
  // Ferrari
  ferrari: "#e8002d",
  scuderia_ferrari: "#e8002d",
  // McLaren
  mclaren: "#ff8000",
  // Aston Martin
  aston_martin: "#358c75",
  aston: "#358c75",
  // Alpine
  alpine: "#ff87bc",
  alpine_renault: "#ff87bc",
};

export const FALLBACK_TEAM_COLOR = "#8888aa";

/**
 * Normalize a constructor identifier or team display name to a lookup key.
 * Lowercases, strips diacritics, replaces non-alphanumeric runs with underscore.
 */
function normalizeKey(value: string): string {
  return value
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[^\p{Letter}\p{Number}]+/gu, "_")
    .replace(/^_+|_+$/g, "");
}

export function getTeamColor(
  constructorId: string | undefined | null
): string {
  if (!constructorId) {
    return FALLBACK_TEAM_COLOR;
  }
  const key = normalizeKey(constructorId);
  if (TEAM_COLORS[key]) {
    return TEAM_COLORS[key];
  }
  // Fallback: try first word (e.g., "Red Bull Racing" → "red")
  // by progressively trimming trailing tokens.
  const tokens = key.split("_");
  for (let i = tokens.length - 1; i > 0; i--) {
    const candidate = tokens.slice(0, i).join("_");
    if (TEAM_COLORS[candidate]) {
      return TEAM_COLORS[candidate];
    }
  }
  return FALLBACK_TEAM_COLOR;
}
