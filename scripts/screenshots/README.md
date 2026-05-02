# Blog Screenshot Capture

Headless Chromium captures of the F1 Race Intelligence UI for blog posts.

## One-time setup

```bash
cd scripts/screenshots
npm install
npx playwright install chromium
```

## Capture screenshots

Against the deployed AKS app (default):

```bash
npm run capture:prod
```

Against a local Vite dev server (run `npm run dev` in `frontend/` first):

```bash
npm run capture:local
```

## Output

Files land in `docs/blog/images/<page>-<viewport>.png`:

- `calendar-desktop.png` / `calendar-mobile.png`
- `standings-desktop.png` / `standings-mobile.png`
- `round-detail-desktop.png` / `round-detail-mobile.png`

## Customizing

Override via env vars:

| Var        | Default                                            | Purpose                               |
|------------|----------------------------------------------------|---------------------------------------|
| `BASE_URL` | `https://f1raceintel.westus3.cloudapp.azure.com`   | Target host                           |
| `OUT_DIR`  | `docs/blog/images` (repo-relative)                 | Where PNGs go                         |
| `ROUND`    | `5`                                                | Which round to capture for detail page |

Example — capture round 3 against local dev:

```bash
BASE_URL=http://localhost:5173 ROUND=3 npm run capture
```

## Adding new pages

Edit the `PAGES` array in [capture.mjs](capture.mjs). Each entry needs a `name`,
a `path`, and a `waitFor` CSS selector that confirms the page rendered.
