<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at specs/004-design-system-brand/plan.md
<!-- SPECKIT END -->

# F1 Race Intelligence Dashboard — Copilot Instructions

## Rule Zero: Constitution Supremacy

The project constitution at `.specify/memory/constitution.md` is the supreme authority.
It overrides all other guidance, including these instructions. Read it at the start of
every session. If any instruction here conflicts with the constitution, the constitution wins.

The five constitutional principles in priority order:
1. **Prescribed Platform Stack** — Go backend, React frontend, Cosmos DB, AKS. No exceptions.
2. **Enforced Three-Tier Boundaries** — UI never calls external APIs. All external integration goes through the backend.
3. **OpenF1 Data Residency and Caching** — Backend caches all OpenF1 data in Cosmos DB before serving clients. No pass-through.
4. **Security and Secrets Baseline** — All secrets in Key Vault via Managed Identity. No plaintext secrets anywhere.
5. **Delivery, Operations, and Dependency Discipline** — Helm charts for K8s. CI/CD: lint → test → build → push → deploy. Structured JSON logs. Minimal dependencies with written justification.

## Essential Context Files

Read these files to understand the project before doing any work:

| File | Purpose |
|------|---------|
| `.specify/memory/constitution.md` | Supreme authority — architecture and governance rules |
| `CONTEXT.md` | Project narrative, origin story, key decisions |
| `README.md` | Current feature status, test counts, deployed URLs, blog post index |
| `specs/{feature}/tasks.md` | Task list for active feature — check what's done and what's next |
| `specs/{feature}/plan.md` | Implementation plan for active feature — architecture, contracts, structure |

## Tech Stack

- **Backend**: Go 1.25+, Chi v5 router, Azure Cosmos DB SDK for Go
- **Frontend**: React 18, TypeScript 5.6, Vite 5.4, Vitest 4.1 (`pool: "threads"`)
- **Infrastructure**: Azure Cosmos DB (serverless), AKS 1.33, ACR (`acrf1raceintel.azurecr.io`), Key Vault (RBAC), nginx ingress
- **Deployment**: Helm v3, GitHub Actions CI/CD (`.github/workflows/ci-cd.yml`)
- **External API**: OpenF1 (`https://api.openf1.org/v1`) — free tier, no API key, rate-limited (~1 req/s)

## Repository Layout

```
backend/           Go API service (Chi router, Cosmos DB, OpenF1 poller)
frontend/          React SPA (Vite, TypeScript, react-router-dom)
deploy/helm/       Helm charts (backend, frontend)
specs/             Spec Kit feature artifacts (spec, plan, tasks per feature)
docs/blog/         Build-in-public blog posts (day-N-slug.md)
infra/             Azure Bicep templates
.specify/          Spec Kit configuration, templates, constitution
.github/           CI/CD workflows, Copilot agents, prompts
```

## Branch Safety (MANDATORY)

**Before executing any feature task, verify the current git branch matches the target feature.**

- If the user says "execute Feature 003 Phase 5," run `git branch --show-current` and confirm the branch name contains `003`.
- If the branch does not match the requested feature number, **STOP and ask the user** before making any changes.
- Never commit feature work to `master` directly — feature work goes on feature branches, then merges to master.
- Never commit feature work to an unrelated feature's branch (e.g., don't put Feature 003 code on a `004-*` branch).
- Blog posts, README updates, and documentation fixes are the only things committed directly to `master`.

Branch naming convention: `{feature-number}-{short-description}` (e.g., `003-race-session-results-phase5-7`)

## Development Conventions

### Commit Messages
Use conventional commits with these prefixes (derived from actual repo history):
`feat:`, `fix:`, `docs:`, `blog:`, `chore:`, `ops:`, `ci:`, `spec:`, `plan:`, `tasks:`
Scoped variants: include the feature number for feature work (`feat(003):`, `feat(Phase 5):`),
and the service or component being fixed for fixes. Known scopes:

| Scope | Covers |
|-------|--------|
| `backend` | General backend Go service |
| `frontend` | General frontend React app |
| `cosmos` | Cosmos DB storage layer (`backend/internal/storage/cosmos/`) |
| `ingest` | OpenF1 poller and session poller (`backend/internal/ingest/`) |
| `calendar` | Calendar API or frontend feature (`backend/internal/api/calendar/`, `frontend/src/features/calendar/`) |
| `rounds` | Rounds API or frontend feature (`backend/internal/api/rounds/`, `frontend/src/features/rounds/`) |
| `standings` | Standings API or frontend feature (`backend/internal/api/standings/`, `frontend/src/features/standings/`) |
| `router` | Chi router wiring (`backend/internal/api/router.go`) |
| `domain` | Domain types and enums (`backend/internal/domain/`) |
| `helm` | Helm charts (`deploy/helm/`) |
| `infra` | Azure Bicep IaC (`infra/bicep/`) |
| `ci` | GitHub Actions workflows (`.github/workflows/`) |
| `lint` | Linter config or formatting fixes |
| `aks` | AKS cluster, ingress, or deployment issues |
| `keyvault` | Key Vault or Managed Identity wiring |

### Build & Test Commands
```bash
# Backend
cd backend && go build ./...
cd backend && go test ./...
export PATH="$HOME/go/bin:$PATH" && cd backend && golangci-lint run ./...

# Frontend
cd frontend && npx vitest run                    # all tests
cd frontend && npx vitest run --reporter=verbose # verbose output
cd frontend && npx tsc --noEmit                  # type check
```

### Frontend Testing Notes
- Vitest config: `pool: "threads"`, `testTimeout: 30000`, `hookTimeout: 30000`
- Use `MemoryRouter` + `Routes` wrapper for components that use `useParams`
- Use `getAllByText` when a value may appear in multiple cells (e.g., team name for multiple drivers)

### Known Gotchas
- **OpenF1 rate limiting**: Add delays between API calls (500ms between sessions, 300ms before driver fetch). Never hammer endpoints in a tight loop.
- **Cosmos DB composite indexes**: Multi-field `ORDER BY` requires composite indexes. Prefer sorting in Go for small result sets.
- **AKS auto-stop**: Cluster stops at 7 PM PT, starts at 8 AM PT via GitHub Actions workflow (`aks-schedule.yml`). Deploys during startup may fail on nginx webhook — retry after cluster is fully ready.
- **WSL paths**: Always quote paths containing spaces in terminal commands.

## Blog Post Workflow

When writing a blog post:
1. Create `docs/blog/day-N-slug.md`
2. Add prev/next navigation links to adjacent blog posts
3. Add the link to `README.md` under the correct feature section in "## Documentation"
4. Update the "## Current Status" section in `README.md` if progress changed
5. Commit blog posts and README updates directly to `master`

### Sensitive data in blog posts (MANDATORY)

Blog posts are public. Before writing or committing a post, redact:

- **Public IP addresses** (home IP, runner IPs, any `a.b.c.d` literal). Replace with `<REDACTED>` or a phrase like "my home IP". This applies to prose, code blocks, and command output transcripts.
- **CIDR ranges** that identify a person or office (e.g. `23.93.233.7/32`). Same treatment.
- **Cluster/tenant/subscription GUIDs**, Key Vault names that aren't already public, storage account keys, connection strings, SAS tokens, bearer tokens, OAuth codes.
- **Personal email addresses, phone numbers, physical addresses.**
- **Internal hostnames or private DNS names** that aren't already public on the deployed site.

Generic public infrastructure that's already discoverable (the deployed site URL, the ACR name `acrf1raceintel.azurecr.io`, the resource group name `rg-f1raceintel`, public GitHub repo URLs, PR numbers) is fine.

When pasting terminal output into a post, scan it line-by-line for the items above and redact before committing. If unsure, redact.

## Session Startup Checklist

At the start of every session, before any implementation work:

- [ ] Read `.specify/memory/constitution.md` — confirm constitutional principles are understood
- [ ] Read `CONTEXT.md` — understand project narrative and key decisions
- [ ] Run `git branch --show-current` — confirm you're on the correct branch for intended work
- [ ] Run `git status` — check for uncommitted changes from a previous session
- [ ] Read `README.md` "Current Status" section — understand what's deployed and what's in progress
- [ ] Read the active feature's `tasks.md` — identify completed tasks and the next task to execute
- [ ] If a feature number is mentioned, verify branch name contains that number (see Branch Safety)
- [ ] Start the AKS cluster if it's not running (it auto-stops at 7 PM PT). Use the GitHub Actions workflow — **do not run `az aks start` directly**:
  1. Trigger via GitHub CLI: `gh workflow run aks-schedule.yml -f action=start`
  2. Wait for it to complete: `gh run watch` (takes ~2-3 minutes)
  3. Then get credentials: `az aks get-credentials --resource-group rg-f1raceintel --name aks-f1raceintel`

## Session Close Checklist

Before ending a session or when the user signals they're done:

- [ ] All tests pass: `cd backend && go test ./...` and `cd frontend && npx vitest run`
- [ ] All changes committed with conventional commit messages
- [ ] Task checkboxes updated in `specs/{feature}/tasks.md` for any completed tasks
- [ ] `README.md` "Current Status" updated if feature progress changed
- [ ] If a blog post was written, it's linked in `README.md` under the correct feature section
- [ ] Summarize what was done, what files changed, and what the next task should be
- [ ] **ASK the user** if they want to stop the AKS cluster to save costs. Do NOT stop it automatically. If they confirm, use the GitHub Actions workflow — **do not run `az aks stop` directly**:
  ```bash
  gh workflow run aks-schedule.yml -f action=stop
  ```
