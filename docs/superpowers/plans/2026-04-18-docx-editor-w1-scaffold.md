# W1 Scaffold (docx-editor platform) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up greenfield infrastructure (monorepo workspace, 8 DB tables, docgen-v2 skeleton, feature flag, package scaffolds) alongside untouched CK5 code so subsequent weeks can fill business logic behind the flag.

**Architecture:** Pure additive scaffold. No CK5 file is modified. New migrations `0101–0108`. New packages under `packages/*`. New Fastify service `apps/docgen-v2` replies only to `/health`. Feature flag `METALDOCS_DOCX_V2_ENABLED` gates everything new. Existing `deploy/compose/docker-compose.yml` already runs MinIO + Gotenberg — reuse.

**Tech Stack:** Go 1.25 · Postgres 16 (raw-SQL migrations mounted via compose) · Node 20 + Fastify + TypeScript · npm workspaces (matches existing npm-based frontend) · Vitest · MinIO (already running) · Gotenberg (already running) · docker compose (additive service only).

**Spec reference:** `docs/superpowers/specs/2026-04-18-docx-editor-platform-design.md` §§ Architecture, Components → Database schema, Out of Scope.

---

## File Structure

**New files (created in this plan):**

```
apps/docgen-v2/
  package.json
  tsconfig.json
  Dockerfile
  .dockerignore
  src/
    index.ts                 # Fastify app + /health
    env.ts                   # zod env parsing
    service-auth.ts          # X-Service-Token middleware
  test/
    health.test.ts

packages/
  shared-types/
    package.json
    tsconfig.json
    src/index.ts             # empty barrel
  shared-tokens/
    package.json
    tsconfig.json
    src/index.ts             # empty barrel
  editor-ui/
    package.json
    tsconfig.json
    src/index.ts             # empty barrel
  form-ui/
    package.json
    tsconfig.json
    src/index.ts             # empty barrel
  docx-editor/
    package.json
    README.md                # fork-trigger doc
    src/index.ts             # empty (reserved for subtree)

internal/modules/
  templates/
    module.go                # empty wiring placeholder
  editor_sessions/
    module.go
  document_revisions/
    module.go

internal/platform/servicebus/
  docgen_v2_client.go        # minimal client (only pings /health)
  docgen_v2_client_test.go

migrations/
  0101_docx_v2_templates.sql
  0102_docx_v2_template_versions.sql
  0103_docx_v2_documents.sql
  0104_docx_v2_editor_sessions.sql
  0105_docx_v2_document_revisions.sql
  0106_docx_v2_autosave_pending_uploads.sql
  0107_docx_v2_document_checkpoints.sql
  0108_docx_v2_template_audit_log.sql

scripts/
  docx-v2-verify-migrations.sh   # bash + psql smoke
  docx-v2-seed-minio.sh          # mc mb tenants bucket

tests/docx_v2/
  scaffold_smoke_test.go     # governance "tests/" rule + scaffold-compiles assertion

docs/runbooks/
  docx-v2-w1-scaffold.md     # governance "runbooks/" rule + operator bring-up

frontend/apps/web/src/features/__tests__/
  featureFlags.docxV2.test.ts  # new subtest; extends existing registry

package.json                 # NEW root package.json with npm workspaces
.env.v2.example              # new env vars documented
docs/superpowers/plans/2026-04-18-docx-editor-w1-scaffold.md  # this file
```

**Modified files:**

```
deploy/compose/docker-compose.yml                     # add docgen-v2 service
internal/platform/config/feature_flags.go             # add DocxV2Enabled field
internal/platform/config/feature_flags_test.go        # add docx_v2 cases (create if missing)
internal/platform/featureflags/handler.go             # expose DOCX_V2_ENABLED in JSON
internal/platform/featureflags/handler_test.go        # assert new field in response
frontend/apps/web/src/features/featureFlags.ts        # add DOCX_V2_ENABLED + isDocxV2Enabled()
apps/api/cmd/metaldocs-api/main.go                    # blank-imports for placeholder modules
.env.example                                          # add new flags
.github/workflows/governance-check.yml                # docx-v2-isolation job
```

**Untouched (invariant):** every file under `frontend/apps/web/src/features/documents/ck5/`, every `service_ck5_*.go`, every `handler_ck5_*.go`, `apps/docgen/`, `apps/ck5-export/`, `apps/ck5-studio/`.

---

## Task 0: Isolated worktree

**Files:**
- N/A (filesystem only)

- [ ] **Step 1: Create worktree and branch off main**

```bash
cd C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs
git worktree add -b feat/docx-v2-w1-scaffold ../MetalDocs-docx-v2-w1 main
cd ../MetalDocs-docx-v2-w1
```

Expected:
```
Preparing worktree (new branch 'feat/docx-v2-w1-scaffold')
HEAD is now at <sha> docs(spec): docx-editor platform design ...
```

- [ ] **Step 2: Verify clean worktree**

```bash
git status
```
Expected: `nothing to commit, working tree clean`.

- [ ] **Step 3: Copy spec into worktree-visible path (if missing)**

Skip — spec already on main; accessible at same relative path in worktree.

---

## Task 1: Root npm workspaces

**Files:**
- Create: `package.json` (root)

- [ ] **Step 1: Write root `package.json` declaring workspaces**

```json
{
  "name": "metaldocs-monorepo",
  "private": true,
  "version": "0.0.0",
  "workspaces": [
    "apps/docgen-v2",
    "packages/*"
  ],
  "scripts": {
    "build:docx-v2": "npm -ws --if-present run build",
    "test:docx-v2": "npm -ws --if-present run test",
    "typecheck:docx-v2": "npm -ws --if-present run typecheck"
  },
  "engines": {
    "node": ">=20.11.0"
  }
}
```

- [ ] **Step 2: Run install, verify no lockfile conflict with `frontend/apps/web`**

```bash
npm install --workspaces --include-workspace-root
```

Expected: `added 0 packages` (all workspaces empty). No error. Root `package-lock.json` appears.

- [ ] **Step 3: Confirm frontend/apps/web untouched**

```bash
git status frontend/apps/web/package-lock.json
```

Expected: `nothing to commit`. `frontend/apps/web` is NOT listed in root workspaces — it manages its own lockfile until W5.

- [ ] **Step 4: Commit**

```bash
rtk git add package.json package-lock.json
rtk git commit -m "chore(docx-v2): introduce root npm workspaces for new packages"
```

---

## Task 2: `packages/shared-types` skeleton

**Files:**
- Create: `packages/shared-types/package.json`
- Create: `packages/shared-types/tsconfig.json`
- Create: `packages/shared-types/src/index.ts`
- Create: `packages/shared-types/test/smoke.test.ts`

- [ ] **Step 1: Write failing smoke test**

`packages/shared-types/test/smoke.test.ts`:
```ts
import { describe, it, expect } from 'vitest';
import * as Types from '../src/index';

describe('@metaldocs/shared-types package', () => {
  it('exports a module object', () => {
    expect(typeof Types).toBe('object');
  });
});
```

- [ ] **Step 2: Write package.json**

`packages/shared-types/package.json`:
```json
{
  "name": "@metaldocs/shared-types",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "scripts": {
    "build": "tsc -p tsconfig.json --noEmit",
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "test": "vitest run"
  },
  "devDependencies": {
    "typescript": "5.4.5",
    "vitest": "1.6.0"
  }
}
```

- [ ] **Step 3: Write tsconfig.json**

`packages/shared-types/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "declaration": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*", "test/**/*"]
}
```

- [ ] **Step 4: Write barrel**

`packages/shared-types/src/index.ts`:
```ts
export {};
```

- [ ] **Step 5: Install + run test**

```bash
npm install --workspace @metaldocs/shared-types
npm run test --workspace @metaldocs/shared-types
```

Expected: `Test Files 1 passed (1) / Tests 1 passed (1)`.

- [ ] **Step 6: Commit**

```bash
rtk git add packages/shared-types package.json package-lock.json
rtk git commit -m "feat(docx-v2): scaffold @metaldocs/shared-types package"
```

---

## Task 3: `packages/shared-tokens` skeleton

**Files:** same shape as Task 2, package name `@metaldocs/shared-tokens`.

- [ ] **Step 1: Write failing smoke test**

`packages/shared-tokens/test/smoke.test.ts`:
```ts
import { describe, it, expect } from 'vitest';
import * as Tokens from '../src/index';

describe('@metaldocs/shared-tokens package', () => {
  it('exports a module object', () => {
    expect(typeof Tokens).toBe('object');
  });
});
```

- [ ] **Step 2: Write package.json**

`packages/shared-tokens/package.json`:
```json
{
  "name": "@metaldocs/shared-tokens",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "scripts": {
    "build": "tsc -p tsconfig.json --noEmit",
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "test": "vitest run"
  },
  "devDependencies": {
    "typescript": "5.4.5",
    "vitest": "1.6.0"
  }
}
```

- [ ] **Step 3: tsconfig.json** — identical content to Task 2 Step 3.

- [ ] **Step 4: Write barrel**

`packages/shared-tokens/src/index.ts`:
```ts
export {};
```

- [ ] **Step 5: Install + test**

```bash
npm install --workspace @metaldocs/shared-tokens
npm run test --workspace @metaldocs/shared-tokens
```

Expected: `Tests 1 passed`.

- [ ] **Step 6: Commit**

```bash
rtk git add packages/shared-tokens package.json package-lock.json
rtk git commit -m "feat(docx-v2): scaffold @metaldocs/shared-tokens package"
```

---

## Task 4: `packages/editor-ui` skeleton

**Files:** package `@metaldocs/editor-ui`.

- [ ] **Step 1: Write failing smoke test**

`packages/editor-ui/test/smoke.test.ts`:
```ts
import { describe, it, expect } from 'vitest';
import * as EditorUI from '../src/index';

describe('@metaldocs/editor-ui package', () => {
  it('exports a module object', () => {
    expect(typeof EditorUI).toBe('object');
  });
});
```

- [ ] **Step 2: Write package.json**

`packages/editor-ui/package.json`:
```json
{
  "name": "@metaldocs/editor-ui",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "scripts": {
    "build": "tsc -p tsconfig.json --noEmit",
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "test": "vitest run"
  },
  "peerDependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@types/react": "18.2.79",
    "@types/react-dom": "18.2.25",
    "react": "18.2.0",
    "react-dom": "18.2.0",
    "typescript": "5.4.5",
    "vitest": "1.6.0"
  }
}
```

- [ ] **Step 3: tsconfig.json** — same as Task 2 plus:
```json
{
  "compilerOptions": {
    "jsx": "react-jsx",
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "declaration": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*", "test/**/*"]
}
```

- [ ] **Step 4: Write barrel**

`packages/editor-ui/src/index.ts`:
```ts
export {};
```

- [ ] **Step 5: Install + test**

```bash
npm install --workspace @metaldocs/editor-ui
npm run test --workspace @metaldocs/editor-ui
```

Expected: `Tests 1 passed`.

- [ ] **Step 6: Commit**

```bash
rtk git add packages/editor-ui package.json package-lock.json
rtk git commit -m "feat(docx-v2): scaffold @metaldocs/editor-ui package"
```

---

## Task 5: `packages/form-ui` skeleton

Identical shape to Task 4, package `@metaldocs/form-ui`.

- [ ] **Step 1: Smoke test** — clone Task 4 Step 1, rename `EditorUI` → `FormUI`, import from `../src/index`.

- [ ] **Step 2: package.json**

`packages/form-ui/package.json`:
```json
{
  "name": "@metaldocs/form-ui",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "types": "./src/index.ts",
  "scripts": {
    "build": "tsc -p tsconfig.json --noEmit",
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "test": "vitest run"
  },
  "peerDependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@types/react": "18.2.79",
    "@types/react-dom": "18.2.25",
    "react": "18.2.0",
    "react-dom": "18.2.0",
    "typescript": "5.4.5",
    "vitest": "1.6.0"
  }
}
```

- [ ] **Step 3: tsconfig.json** — identical to Task 4 Step 3.

- [ ] **Step 4: Barrel** — `export {};`.

- [ ] **Step 5: Install + test**

```bash
npm install --workspace @metaldocs/form-ui
npm run test --workspace @metaldocs/form-ui
```
Expected: `Tests 1 passed`.

- [ ] **Step 6: Commit**

```bash
rtk git add packages/form-ui package.json package-lock.json
rtk git commit -m "feat(docx-v2): scaffold @metaldocs/form-ui package"
```

---

## Task 6: `packages/docx-editor` fork scaffold

**Files:**
- Create: `packages/docx-editor/package.json`
- Create: `packages/docx-editor/README.md`
- Create: `packages/docx-editor/src/index.ts`

- [ ] **Step 1: Write package.json**

```json
{
  "name": "@metaldocs/docx-editor-fork",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./src/index.ts",
  "scripts": {
    "build": "echo 'noop: reserved for subtree fork'",
    "test": "echo 'noop: reserved for subtree fork'"
  }
}
```

- [ ] **Step 2: Write README.md**

```markdown
# packages/docx-editor (reserved)

Empty scaffold for a future subtree fork of `@eigenpal/docx-js-editor`.

**Fork trigger:** first confirmed blocker requiring library internals
(restricted-cell editing, custom node schemas, paginator override).
Until then, `@metaldocs/editor-ui` depends on the upstream package
pinned at `0.0.34` exact.

To fork:

```bash
git subtree add --prefix=packages/docx-editor \
  https://github.com/eigenpal/docx-js-editor v0.0.34 --squash
```
```

- [ ] **Step 3: Write barrel**

`packages/docx-editor/src/index.ts`:
```ts
export {};
```

- [ ] **Step 4: Verify workspace recognizes package**

```bash
npm ls --workspaces --depth 0
```

Expected: output lists `@metaldocs/docx-editor-fork@0.0.0`.

- [ ] **Step 5: Commit**

```bash
rtk git add packages/docx-editor package.json package-lock.json
rtk git commit -m "feat(docx-v2): reserve packages/docx-editor subtree scaffold"
```

---

## Task 7: `apps/docgen-v2` Fastify skeleton with /health

**Files:**
- Create: `apps/docgen-v2/package.json`
- Create: `apps/docgen-v2/tsconfig.json`
- Create: `apps/docgen-v2/src/env.ts`
- Create: `apps/docgen-v2/src/service-auth.ts`
- Create: `apps/docgen-v2/src/index.ts`
- Create: `apps/docgen-v2/test/health.test.ts`

- [ ] **Step 1: Write failing health test**

`apps/docgen-v2/test/health.test.ts`:
```ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { buildApp } from '../src/index';
import type { FastifyInstance } from 'fastify';

let app: FastifyInstance;

beforeAll(async () => {
  // env validator in env.ts requires token >= 16 chars
  process.env.DOCGEN_V2_SERVICE_TOKEN = 'test-token-0123456789';
  process.env.DOCGEN_V2_PORT = '0';
  app = await buildApp();
});

afterAll(async () => {
  await app.close();
});

describe('GET /health', () => {
  it('returns 200 and version/status payload', async () => {
    const res = await app.inject({ method: 'GET', url: '/health' });
    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.status).toBe('ok');
    expect(typeof body.version).toBe('string');
  });

  it('does NOT require X-Service-Token on /health', async () => {
    const res = await app.inject({ method: 'GET', url: '/health' });
    expect(res.statusCode).toBe(200);
  });

  it('rejects requests to other paths without X-Service-Token', async () => {
    const res = await app.inject({ method: 'POST', url: '/render/docx', payload: {} });
    expect(res.statusCode).toBe(401);
  });
});
```

- [ ] **Step 2: Write package.json**

```json
{
  "name": "@metaldocs/docgen-v2",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./dist/index.js",
  "scripts": {
    "build": "tsc -p tsconfig.json",
    "start": "node dist/index.js",
    "dev": "tsx watch src/index.ts",
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "test": "vitest run"
  },
  "dependencies": {
    "fastify": "4.26.2",
    "zod": "3.23.8"
  },
  "devDependencies": {
    "@types/node": "20.12.12",
    "tsx": "4.11.0",
    "typescript": "5.4.5",
    "vitest": "1.6.0"
  }
}
```

- [ ] **Step 3: Write tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "declaration": false,
    "outDir": "./dist",
    "rootDir": "./src",
    "isolatedModules": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["test/**/*"]
}
```

- [ ] **Step 4: Write env.ts**

```ts
import { z } from 'zod';

const EnvSchema = z.object({
  DOCGEN_V2_PORT: z.coerce.number().int().min(0).max(65535).default(3100),
  DOCGEN_V2_SERVICE_TOKEN: z.string().min(16, 'service token must be >= 16 chars'),
  DOCGEN_V2_LOG_LEVEL: z.enum(['fatal','error','warn','info','debug','trace']).default('info'),
  DOCGEN_V2_VERSION: z.string().default('0.0.0-dev'),
});

export type Env = z.infer<typeof EnvSchema>;

export function loadEnv(): Env {
  const parsed = EnvSchema.safeParse(process.env);
  if (!parsed.success) {
    const flat = parsed.error.flatten().fieldErrors;
    throw new Error(`invalid env: ${JSON.stringify(flat)}`);
  }
  return parsed.data;
}
```

- [ ] **Step 5: Write service-auth.ts**

```ts
import type { FastifyInstance } from 'fastify';

export function registerServiceAuth(app: FastifyInstance, token: string): void {
  app.addHook('onRequest', async (req, reply) => {
    if (req.url === '/health') return;
    const header = req.headers['x-service-token'];
    if (typeof header !== 'string' || header !== token) {
      reply.code(401).send({ error: 'unauthorized' });
    }
  });
}
```

- [ ] **Step 6: Write index.ts**

```ts
import Fastify, { type FastifyInstance } from 'fastify';
import { loadEnv } from './env';
import { registerServiceAuth } from './service-auth';

export async function buildApp(): Promise<FastifyInstance> {
  const env = loadEnv();
  const app = Fastify({ logger: { level: env.DOCGEN_V2_LOG_LEVEL } });

  registerServiceAuth(app, env.DOCGEN_V2_SERVICE_TOKEN);

  app.get('/health', async () => ({ status: 'ok', version: env.DOCGEN_V2_VERSION }));

  return app;
}

// Only start when run as entrypoint (not under test import).
if (import.meta.url === `file://${process.argv[1]}`) {
  const env = loadEnv();
  buildApp().then((app) => {
    app.listen({ port: env.DOCGEN_V2_PORT, host: '0.0.0.0' })
       .catch((err) => { app.log.fatal(err); process.exit(1); });
  });
}
```

- [ ] **Step 7: Install + run test**

```bash
npm install --workspace @metaldocs/docgen-v2
npm run test --workspace @metaldocs/docgen-v2
```

Expected: `Tests 3 passed`.

- [ ] **Step 8: Build sanity**

```bash
npm run build --workspace @metaldocs/docgen-v2
```

Expected: zero TS errors, `dist/index.js` appears.

- [ ] **Step 9: Commit**

```bash
rtk git add apps/docgen-v2 package.json package-lock.json
rtk git commit -m "feat(docgen-v2): Fastify skeleton with /health and X-Service-Token"
```

---

## Task 8: Dockerfile + docker-compose service for docgen-v2

**Files:**
- Create: `apps/docgen-v2/Dockerfile`
- Create: `apps/docgen-v2/.dockerignore`
- Modify: `deploy/compose/docker-compose.yml` (add service only)

- [ ] **Step 1: Write Dockerfile**

```dockerfile
FROM node:20.11-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
COPY apps/docgen-v2/package.json ./apps/docgen-v2/
RUN npm ci --workspace @metaldocs/docgen-v2 --include-workspace-root
COPY apps/docgen-v2 ./apps/docgen-v2
RUN npm run build --workspace @metaldocs/docgen-v2

FROM node:20.11-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/apps/docgen-v2/dist ./dist
COPY --from=build /app/apps/docgen-v2/package.json ./package.json
USER node
EXPOSE 3100
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3100/health || exit 1
CMD ["node", "dist/index.js"]
```

- [ ] **Step 2: Write .dockerignore**

```
node_modules
dist
test
*.log
.env*
```

- [ ] **Step 3: Read current compose to find insertion point**

```bash
rtk read deploy/compose/docker-compose.yml
```
Note line where `gotenberg:` service ends and next service begins — insert `docgen-v2:` after `gotenberg:`.

- [ ] **Step 4: Edit docker-compose.yml to add docgen-v2 service**

Add block after the `gotenberg:` service (indentation must match sibling services; 2 spaces):

```yaml
  docgen-v2:
    build:
      context: ../..
      dockerfile: apps/docgen-v2/Dockerfile
    environment:
      DOCGEN_V2_PORT: 3100
      DOCGEN_V2_SERVICE_TOKEN: ${DOCGEN_V2_SERVICE_TOKEN:?DOCGEN_V2_SERVICE_TOKEN required}
      DOCGEN_V2_LOG_LEVEL: info
      DOCGEN_V2_VERSION: ${DOCGEN_V2_VERSION:-dev}
    ports:
      - "3100:3100"
    depends_on:
      minio:
        condition: service_started
      gotenberg:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:3100/health"]
      interval: 30s
      timeout: 3s
      retries: 3
```

MinIO has no healthcheck in the existing compose file, and introducing one is out of scope for W1 (it would belong to a MinIO-ops runbook change). `service_started` matches the existing `api` service pattern. Gotenberg already has a healthcheck so we keep `service_healthy` there.

- [ ] **Step 5: Validate compose file syntax**

```bash
docker compose -f deploy/compose/docker-compose.yml config > /dev/null
```

Expected: no output; exit 0.

- [ ] **Step 6: Build the image**

```bash
DOCGEN_V2_SERVICE_TOKEN=0123456789abcdef \
  docker compose -f deploy/compose/docker-compose.yml build docgen-v2
```

Expected: build succeeds, final image tagged.

- [ ] **Step 7: Start service, hit /health**

```bash
DOCGEN_V2_SERVICE_TOKEN=0123456789abcdef \
  docker compose -f deploy/compose/docker-compose.yml up -d docgen-v2
sleep 5
curl -s http://127.0.0.1:3100/health
```

Expected: `{"status":"ok","version":"dev"}`.

- [ ] **Step 8: Tear down**

```bash
docker compose -f deploy/compose/docker-compose.yml stop docgen-v2
```

- [ ] **Step 9: Commit**

```bash
rtk git add apps/docgen-v2/Dockerfile apps/docgen-v2/.dockerignore deploy/compose/docker-compose.yml
rtk git commit -m "feat(docgen-v2): Dockerfile and compose service wiring"
```

---

## Task 9: Env var plumbing (`.env.example` + `.env.v2.example`)

**Files:**
- Create: `.env.v2.example`
- Modify: `.env.example` (append new block)

- [ ] **Step 1: Write `.env.v2.example`**

```
# docgen-v2 + docx-editor platform (W1 scaffold)
DOCGEN_V2_SERVICE_TOKEN=please-change-me-minimum-16-chars
DOCGEN_V2_PORT=3100
DOCGEN_V2_LOG_LEVEL=info
DOCGEN_V2_VERSION=dev

# Go API → docgen-v2 client
METALDOCS_DOCGEN_V2_URL=http://docgen-v2:3100
METALDOCS_DOCGEN_V2_SERVICE_TOKEN=please-change-me-minimum-16-chars

# Feature flag (per-tenant via DB/config; this is the global default)
METALDOCS_DOCX_V2_ENABLED=false
```

- [ ] **Step 2: Append to `.env.example`**

```bash
rtk read .env.example
```
Then Edit: append after the existing last non-empty line:
```

# --- docx-editor platform (W1 scaffold) ---
# See .env.v2.example for full block.
DOCGEN_V2_SERVICE_TOKEN=
METALDOCS_DOCGEN_V2_URL=http://docgen-v2:3100
METALDOCS_DOCGEN_V2_SERVICE_TOKEN=
METALDOCS_DOCX_V2_ENABLED=false
```

- [ ] **Step 3: Commit**

```bash
rtk git add .env.example .env.v2.example
rtk git commit -m "chore(docx-v2): document new env vars (service token, flag, URL)"
```

---

## Task 10: Migration 0101 — `templates`

**Files:**
- Create: `migrations/0101_docx_v2_templates.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0101_docx_v2_templates.sql
-- Logical template (e.g. "Purchase Order"). Owned by a tenant.
-- Part of docx-editor platform (W1 scaffold). Tables in this block
-- are prefixed docx_v2_ in the migration filename but take their
-- spec names as the table identifier because they supersede CK5.

BEGIN;

CREATE TABLE IF NOT EXISTS templates (
  id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                     UUID NOT NULL,
  key                           TEXT NOT NULL,
  name                          TEXT NOT NULL,
  description                   TEXT,
  current_published_version_id  UUID,
  created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by                    UUID NOT NULL,
  CONSTRAINT templates_tenant_key_unique UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_templates_tenant
  ON templates (tenant_id);

COMMIT;
```

- [ ] **Step 2: Apply to running local postgres**

```bash
docker compose -f deploy/compose/docker-compose.yml up -d postgres
sleep 3
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0101_docx_v2_templates.sql
```

Expected: `BEGIN / CREATE TABLE / CREATE INDEX / COMMIT`.

- [ ] **Step 3: Verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d templates"
```

Expected: columns match DDL; unique constraint on `(tenant_id, key)` shown.

- [ ] **Step 4: Commit**

```bash
rtk git add migrations/0101_docx_v2_templates.sql
rtk git commit -m "feat(docx-v2): migration 0101 templates"
```

---

## Task 11: Migration 0102 — `template_versions`

**Files:**
- Create: `migrations/0102_docx_v2_template_versions.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0102_docx_v2_template_versions.sql
BEGIN;

CREATE TABLE IF NOT EXISTS template_versions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id           UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
  version_num           INT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','published','deprecated')),
  grammar_version       INT NOT NULL DEFAULT 1,
  docx_storage_key      TEXT NOT NULL,
  schema_storage_key    TEXT NOT NULL,
  docx_content_hash     TEXT NOT NULL,
  schema_content_hash   TEXT NOT NULL,
  published_at          TIMESTAMPTZ,
  published_by          UUID,
  deprecated_at         TIMESTAMPTZ,
  lock_version          INT NOT NULL DEFAULT 0,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL,
  CONSTRAINT template_versions_template_num_unique UNIQUE (template_id, version_num)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_draft_per_template
  ON template_versions (template_id) WHERE status = 'draft';

ALTER TABLE templates
  ADD CONSTRAINT fk_templates_current_published
    FOREIGN KEY (current_published_version_id)
    REFERENCES template_versions(id)
    DEFERRABLE INITIALLY IMMEDIATE;

COMMIT;
```

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0102_docx_v2_template_versions.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d template_versions"
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "SELECT indexname FROM pg_indexes WHERE tablename='template_versions';"
```

Expected: table + partial unique index `idx_one_draft_per_template` present.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0102_docx_v2_template_versions.sql
rtk git commit -m "feat(docx-v2): migration 0102 template_versions + draft uniqueness"
```

---

## Task 12: Migration 0103 — `documents`

**Files:**
- Create: `migrations/0103_docx_v2_documents.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0103_docx_v2_documents.sql
BEGIN;

CREATE TABLE IF NOT EXISTS documents_v2 (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL,
  template_version_id   UUID NOT NULL REFERENCES template_versions(id),
  name                  TEXT NOT NULL,
  status                TEXT NOT NULL CHECK (status IN ('draft','finalized','archived')),
  form_data_json        JSONB NOT NULL,
  current_revision_id   UUID,
  active_session_id     UUID,
  finalized_at          TIMESTAMPTZ,
  archived_at           TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by            UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_documents_v2_tenant_status
  ON documents_v2 (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_documents_v2_template_version
  ON documents_v2 (template_version_id);
CREATE INDEX IF NOT EXISTS idx_documents_v2_form_data_gin
  ON documents_v2 USING GIN (form_data_json jsonb_path_ops);

COMMIT;
```

Table is named `documents_v2` during W1–W4 to avoid collision with legacy `documents`. Renamed to `documents` at W5 cutover (plan E).

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0103_docx_v2_documents.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d documents_v2"
```

Expected: 3 indexes listed; `tenant_id`, `form_data_json` JSONB present.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0103_docx_v2_documents.sql
rtk git commit -m "feat(docx-v2): migration 0103 documents_v2 (renamed to documents at W5)"
```

---

## Task 13: Migration 0104 — `editor_sessions`

**Files:**
- Create: `migrations/0104_docx_v2_editor_sessions.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0104_docx_v2_editor_sessions.sql
BEGIN;

CREATE TABLE IF NOT EXISTS editor_sessions (
  id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id                     UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  user_id                         UUID NOT NULL,
  acquired_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at                      TIMESTAMPTZ NOT NULL,
  released_at                     TIMESTAMPTZ,
  last_acknowledged_revision_id   UUID NOT NULL,
  status                          TEXT NOT NULL CHECK (status IN ('active','expired','released','force_released'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_active_session_per_doc
  ON editor_sessions (document_id) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_editor_sessions_expires
  ON editor_sessions (expires_at) WHERE status = 'active';

COMMIT;
```

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0104_docx_v2_editor_sessions.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d editor_sessions"
```

Expected: two partial indexes listed.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0104_docx_v2_editor_sessions.sql
rtk git commit -m "feat(docx-v2): migration 0104 editor_sessions + partial indexes"
```

---

## Task 14: Migration 0105 — `document_revisions`

**Files:**
- Create: `migrations/0105_docx_v2_document_revisions.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0105_docx_v2_document_revisions.sql
BEGIN;

CREATE TABLE IF NOT EXISTS document_revisions (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id            UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  revision_num           BIGSERIAL,
  parent_revision_id     UUID REFERENCES document_revisions(id),
  session_id             UUID NOT NULL REFERENCES editor_sessions(id),
  storage_key            TEXT NOT NULL,
  content_hash           TEXT NOT NULL,
  form_data_snapshot     JSONB,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT document_revisions_doc_hash_unique UNIQUE (document_id, content_hash)
);

CREATE INDEX IF NOT EXISTS idx_revisions_doc_num
  ON document_revisions (document_id, revision_num DESC);

ALTER TABLE documents_v2
  ADD CONSTRAINT fk_documents_v2_current_revision
    FOREIGN KEY (current_revision_id) REFERENCES document_revisions(id)
    DEFERRABLE INITIALLY IMMEDIATE,
  ADD CONSTRAINT fk_documents_v2_active_session
    FOREIGN KEY (active_session_id) REFERENCES editor_sessions(id)
    DEFERRABLE INITIALLY IMMEDIATE;

COMMIT;
```

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0105_docx_v2_document_revisions.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d document_revisions"
```

Expected: BIGSERIAL column, unique `(document_id, content_hash)`, FK back to self.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0105_docx_v2_document_revisions.sql
rtk git commit -m "feat(docx-v2): migration 0105 document_revisions (content-addressed)"
```

---

## Task 15: Migration 0106 — `autosave_pending_uploads`

**Files:**
- Create: `migrations/0106_docx_v2_autosave_pending_uploads.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0106_docx_v2_autosave_pending_uploads.sql
BEGIN;

CREATE TABLE IF NOT EXISTS autosave_pending_uploads (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id           UUID NOT NULL REFERENCES editor_sessions(id) ON DELETE CASCADE,
  document_id          UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  base_revision_id     UUID NOT NULL REFERENCES document_revisions(id),
  content_hash         TEXT NOT NULL,
  storage_key          TEXT NOT NULL,
  presigned_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at           TIMESTAMPTZ NOT NULL,
  consumed_at          TIMESTAMPTZ,
  CONSTRAINT autosave_pending_uniq
    UNIQUE (session_id, base_revision_id, content_hash)
);

CREATE INDEX IF NOT EXISTS idx_pending_expired
  ON autosave_pending_uploads (expires_at) WHERE consumed_at IS NULL;

COMMIT;
```

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0106_docx_v2_autosave_pending_uploads.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d autosave_pending_uploads"
```

Expected: composite unique `(session_id, base_revision_id, content_hash)` present.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0106_docx_v2_autosave_pending_uploads.sql
rtk git commit -m "feat(docx-v2): migration 0106 autosave_pending_uploads"
```

---

## Task 16: Migration 0107 — `document_checkpoints`

**Files:**
- Create: `migrations/0107_docx_v2_document_checkpoints.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0107_docx_v2_document_checkpoints.sql
BEGIN;

CREATE TABLE IF NOT EXISTS document_checkpoints (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id        UUID NOT NULL REFERENCES documents_v2(id) ON DELETE CASCADE,
  revision_id        UUID NOT NULL REFERENCES document_revisions(id),
  version_num        INT NOT NULL,
  label              TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by         UUID NOT NULL,
  CONSTRAINT document_checkpoints_doc_num_unique UNIQUE (document_id, version_num)
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_doc
  ON document_checkpoints (document_id, version_num DESC);

COMMIT;
```

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0107_docx_v2_document_checkpoints.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\d document_checkpoints"
```

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0107_docx_v2_document_checkpoints.sql
rtk git commit -m "feat(docx-v2): migration 0107 document_checkpoints"
```

---

## Task 17: Migration 0108 — `template_audit_log` (append-only grants)

**Files:**
- Create: `migrations/0108_docx_v2_template_audit_log.sql`

- [ ] **Step 1: Write migration**

```sql
-- 0108_docx_v2_template_audit_log.sql
BEGIN;

CREATE TABLE IF NOT EXISTS template_audit_log (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL,
  template_id          UUID,
  template_version_id  UUID,
  document_id          UUID,
  action               TEXT NOT NULL,
  actor_user_id        UUID NOT NULL,
  metadata_json        JSONB,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_created
  ON template_audit_log (tenant_id, created_at DESC);

-- Append-only enforcement at DB layer.
-- Runtime role name is configured via METALDOCS_DB_APP_ROLE env; default 'metaldocs_app'.
DO $$
DECLARE role_name TEXT := current_setting('metaldocs.app_role', true);
BEGIN
  IF role_name IS NULL OR role_name = '' THEN
    role_name := 'metaldocs_app';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name) THEN
    EXECUTE format('REVOKE UPDATE, DELETE ON template_audit_log FROM %I', role_name);
    EXECUTE format('GRANT  INSERT, SELECT ON template_audit_log TO %I', role_name);
  END IF;
END$$;

COMMIT;
```

- [ ] **Step 2: Apply + verify**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} < migrations/0108_docx_v2_template_audit_log.sql
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -c "\dp template_audit_log"
```

Expected: `INSERT/SELECT` granted to app role if role exists; no UPDATE/DELETE listed.

- [ ] **Step 3: Commit**

```bash
rtk git add migrations/0108_docx_v2_template_audit_log.sql
rtk git commit -m "feat(docx-v2): migration 0108 template_audit_log (append-only grants)"
```

---

## Task 18: Migration smoke script

**Files:**
- Create: `scripts/docx-v2-verify-migrations.sh`

- [ ] **Step 1: Write script**

```bash
#!/usr/bin/env bash
set -euo pipefail

# Verifies all docx-v2 migrations applied and all tables+indexes present.

SERVICE="${SERVICE:-postgres}"
COMPOSE_FILE="${COMPOSE_FILE:-deploy/compose/docker-compose.yml}"
DB_USER="${PGUSER:-metaldocs}"
DB_NAME="${PGDATABASE:-metaldocs}"

expected_tables=(
  templates
  template_versions
  documents_v2
  editor_sessions
  document_revisions
  autosave_pending_uploads
  document_checkpoints
  template_audit_log
)

expected_indexes=(
  idx_one_draft_per_template
  idx_one_active_session_per_doc
  idx_pending_expired
  idx_documents_v2_form_data_gin
)

psql_exec() {
  docker compose -f "$COMPOSE_FILE" exec -T "$SERVICE" \
    psql -U "$DB_USER" -d "$DB_NAME" -tAc "$1"
}

fail=0
for t in "${expected_tables[@]}"; do
  got=$(psql_exec "SELECT to_regclass('public.$t') IS NOT NULL")
  if [[ "$got" != "t" ]]; then
    echo "MISSING table: $t"
    fail=1
  fi
done

for idx in "${expected_indexes[@]}"; do
  got=$(psql_exec "SELECT COUNT(*) FROM pg_indexes WHERE indexname='$idx'")
  if [[ "$got" == "0" ]]; then
    echo "MISSING index: $idx"
    fail=1
  fi
done

if [[ "$fail" == "1" ]]; then
  echo "FAIL"
  exit 1
fi

echo "OK: all 8 tables + 4 critical indexes present"
```

- [ ] **Step 2: Make executable + run**

```bash
chmod +x scripts/docx-v2-verify-migrations.sh
bash scripts/docx-v2-verify-migrations.sh
```

Expected: `OK: all 8 tables + 4 critical indexes present`.

- [ ] **Step 3: Commit**

```bash
rtk git add scripts/docx-v2-verify-migrations.sh
rtk git commit -m "chore(docx-v2): migration smoke verification script"
```

---

## Task 19: Feature flag Go side (config + handler wiring)

**Files:**
- Modify: `internal/platform/config/feature_flags.go` (new field + env loader)
- Create: `internal/platform/config/feature_flags_test.go` (if not already present; add docx_v2 cases either way)
- Modify: `internal/platform/featureflags/handler.go` (expose flag in JSON response)
- Modify: `internal/platform/featureflags/handler_test.go` (if exists; else create)

**Why in-place (not new package):** the existing project has `internal/platform/config/feature_flags.go` for loading and `internal/platform/featureflags/handler.go` for exposing flags to the frontend. Introducing a third package (`feature_flags`) would diverge from convention and leave the flag unreachable by the `GET /api/v1/feature-flags` frontend fetch.

- [ ] **Step 1: Read existing files**

```bash
rtk read internal/platform/config/feature_flags.go
rtk read internal/platform/featureflags/handler.go
```

Note the exact struct name (`FeatureFlagsConfig`), the existing field (`MDDMNativeExportRolloutPercent`), the env read call, and the response struct (`featureFlagsResponse` with `MDDMNativeExportRolloutPct` field tagged `json:"MDDM_NATIVE_EXPORT_ROLLOUT_PCT"`).

- [ ] **Step 2: Write failing config test**

Append to `internal/platform/config/feature_flags_test.go` (create if missing):

```go
package config_test

import (
	"testing"

	"metaldocs/internal/platform/config"
)

func TestDocxV2Enabled_Default(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "")
	cfg := config.LoadFeatureFlagsConfig()
	if cfg.DocxV2Enabled {
		t.Fatalf("expected default false, got true")
	}
}

func TestDocxV2Enabled_True(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "true")
	cfg := config.LoadFeatureFlagsConfig()
	if !cfg.DocxV2Enabled {
		t.Fatalf("expected true")
	}
}

func TestDocxV2Enabled_False(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "false")
	cfg := config.LoadFeatureFlagsConfig()
	if cfg.DocxV2Enabled {
		t.Fatalf("expected false")
	}
}

func TestDocxV2Enabled_Unknown(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "notabool")
	cfg := config.LoadFeatureFlagsConfig()
	if cfg.DocxV2Enabled {
		t.Fatalf("unknown must default to false")
	}
}
```

(If `LoadFeatureFlagsConfig` is named differently in the repo — e.g. `LoadFeatureFlags` — use the existing name, exact.)

- [ ] **Step 3: Run test (FAIL)**

```bash
go test ./internal/platform/config/...
```

Expected: `undefined: cfg.DocxV2Enabled` or similar.

- [ ] **Step 4: Modify `feature_flags.go` struct + loader**

Add field to `FeatureFlagsConfig`:

```go
// DocxV2Enabled gates the docx-editor v2 platform (W1-W5 rollout).
// Defaults to false. Read from METALDOCS_DOCX_V2_ENABLED.
DocxV2Enabled bool
```

Add helper (file-local):

```go
func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "true" || v == "1" || v == "yes"
}
```

In the existing loader (e.g. `LoadFeatureFlagsConfig`), populate:

```go
cfg.DocxV2Enabled = envBool("METALDOCS_DOCX_V2_ENABLED")
```

Add `"os"` / `"strings"` imports if missing.

- [ ] **Step 5: Run test (PASS)**

```bash
go test ./internal/platform/config/... -v
```

Expected: 4 new subtests PASS, existing tests untouched.

- [ ] **Step 6: Write failing handler test**

Append to `internal/platform/featureflags/handler_test.go` (create if missing):

```go
package featureflags_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/featureflags"
)

func TestHandler_IncludesDocxV2Enabled(t *testing.T) {
	h := featureflags.NewHandler(config.FeatureFlagsConfig{DocxV2Enabled: true})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["DOCX_V2_ENABLED"] != true {
		t.Fatalf("expected DOCX_V2_ENABLED=true in response, got %v", body["DOCX_V2_ENABLED"])
	}
}

func TestHandler_DocxV2Disabled_DefaultFalse(t *testing.T) {
	h := featureflags.NewHandler(config.FeatureFlagsConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)

	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["DOCX_V2_ENABLED"] != false {
		t.Fatalf("expected DOCX_V2_ENABLED=false, got %v", body["DOCX_V2_ENABLED"])
	}
}
```

- [ ] **Step 7: Run handler test (FAIL — response struct missing field)**

```bash
go test ./internal/platform/featureflags/...
```

- [ ] **Step 8: Extend `handler.go` response struct**

Modify `featureFlagsResponse`:

```go
type featureFlagsResponse struct {
	MDDMNativeExportRolloutPct int  `json:"MDDM_NATIVE_EXPORT_ROLLOUT_PCT"`
	DocxV2Enabled              bool `json:"DOCX_V2_ENABLED"`
}
```

Populate in `handle`:

```go
_ = json.NewEncoder(w).Encode(featureFlagsResponse{
	MDDMNativeExportRolloutPct: h.cfg.MDDMNativeExportRolloutPercent,
	DocxV2Enabled:              h.cfg.DocxV2Enabled,
})
```

- [ ] **Step 9: Run handler test (PASS)**

```bash
go test ./internal/platform/featureflags/... -v
```

Expected: both new subtests PASS.

- [ ] **Step 10: Commit**

```bash
rtk git add internal/platform/config/feature_flags.go internal/platform/config/feature_flags_test.go internal/platform/featureflags/handler.go internal/platform/featureflags/handler_test.go
rtk git commit -m "feat(docx-v2): DocxV2Enabled flag in config + /api/v1/feature-flags response"
```

---

## Task 20: Feature flag frontend side (extend canonical registry)

**Files:**
- Modify: `frontend/apps/web/src/features/featureFlags.ts`
- Create: `frontend/apps/web/src/features/__tests__/featureFlags.docxV2.test.ts`

**Why extend, not a new file:** the existing `featureFlags.ts` is the single source of truth — it reads both `window.__METALDOCS_FEATURE_FLAGS` AND overrides from `GET /api/v1/feature-flags` via `initFeatureFlags()`. A standalone window-only reader would miss the fetch path.

- [ ] **Step 1: Read existing `featureFlags.ts`** (already shown; confirm `FeatureFlags` type, `readFlags`, `initFeatureFlags`, and the window reader shape).

- [ ] **Step 2: Write failing test**

`frontend/apps/web/src/features/__tests__/featureFlags.docxV2.test.ts`:
```ts
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('DOCX_V2_ENABLED flag', () => {
  beforeEach(() => {
    vi.resetModules();
    (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Record<string, unknown> })
      .__METALDOCS_FEATURE_FLAGS = undefined;
  });

  it('defaults to false when no source provides it', async () => {
    const { featureFlags, isDocxV2Enabled } = await import('../featureFlags');
    expect(featureFlags.DOCX_V2_ENABLED).toBe(false);
    expect(isDocxV2Enabled()).toBe(false);
  });

  it('reads true from window injection', async () => {
    (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Record<string, unknown> })
      .__METALDOCS_FEATURE_FLAGS = { DOCX_V2_ENABLED: true };
    const { featureFlags, isDocxV2Enabled } = await import('../featureFlags');
    expect(featureFlags.DOCX_V2_ENABLED).toBe(true);
    expect(isDocxV2Enabled()).toBe(true);
  });

  it('treats non-boolean truthy as false (strict)', async () => {
    (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Record<string, unknown> })
      .__METALDOCS_FEATURE_FLAGS = { DOCX_V2_ENABLED: 'true' };
    const { featureFlags } = await import('../featureFlags');
    expect(featureFlags.DOCX_V2_ENABLED).toBe(false);
  });

  it('initFeatureFlags patches from /api/v1/feature-flags', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ DOCX_V2_ENABLED: true }), { status: 200 })
    );
    const mod = await import('../featureFlags');
    await mod.initFeatureFlags();
    expect(mod.featureFlags.DOCX_V2_ENABLED).toBe(true);
    fetchSpy.mockRestore();
  });
});
```

- [ ] **Step 3: Run test (FAIL — field absent)**

```bash
cd frontend/apps/web && npm run test -- featureFlags.docxV2.test.ts
```

- [ ] **Step 4: Extend `featureFlags.ts`**

Modify the `FeatureFlags` type:

```ts
type FeatureFlags = {
  MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
  MDDM_NATIVE_EXPORT: boolean;
  /** docx-editor v2 platform gate. Strict boolean; default false. */
  DOCX_V2_ENABLED: boolean;
};
```

Extend `readWindowFlags` signature + body to include `DOCX_V2_ENABLED`:

```ts
function readWindowFlags():
  | Partial<{ MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number; DOCX_V2_ENABLED: boolean }>
  | undefined {
  if (typeof window === "undefined") return undefined;
  return (
    window as unknown as {
      __METALDOCS_FEATURE_FLAGS?: Partial<{
        MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
        DOCX_V2_ENABLED: boolean;
      }>;
    }
  ).__METALDOCS_FEATURE_FLAGS;
}
```

Add helper + default in `readFlags`:

```ts
function strictBool(raw: unknown): boolean {
  return raw === true;
}

function readFlags(): FeatureFlags {
  const injected = readWindowFlags();
  return {
    MDDM_NATIVE_EXPORT_ROLLOUT_PCT: clampPct(injected?.MDDM_NATIVE_EXPORT_ROLLOUT_PCT),
    MDDM_NATIVE_EXPORT: false,
    DOCX_V2_ENABLED: strictBool(injected?.DOCX_V2_ENABLED),
  };
}
```

Patch `initFeatureFlags` to overwrite `DOCX_V2_ENABLED`:

```ts
export async function initFeatureFlags(): Promise<void> {
  try {
    const res = await fetch("/api/v1/feature-flags");
    if (!res.ok) return;
    const data = (await res.json()) as Partial<{
      MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
      DOCX_V2_ENABLED: boolean;
    }>;
    featureFlags.MDDM_NATIVE_EXPORT_ROLLOUT_PCT = clampPct(data.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
    featureFlags.DOCX_V2_ENABLED = strictBool(data.DOCX_V2_ENABLED);
  } catch {
    // keep defaults
  }
}
```

Append convenience export:

```ts
/** True iff the docx-editor v2 platform is active for this session. */
export function isDocxV2Enabled(): boolean {
  return featureFlags.DOCX_V2_ENABLED;
}
```

- [ ] **Step 5: Run test (PASS)**

```bash
cd frontend/apps/web && npm run test -- featureFlags.docxV2.test.ts
```

Expected: 4/4 pass.

- [ ] **Step 6: Run full web test suite to catch regression**

```bash
cd frontend/apps/web && npm run test
```

Expected: existing MDDM flag tests still pass.

- [ ] **Step 7: Commit**

```bash
rtk git add frontend/apps/web/src/features/featureFlags.ts frontend/apps/web/src/features/__tests__/featureFlags.docxV2.test.ts
rtk git commit -m "feat(docx-v2): DOCX_V2_ENABLED in frontend registry + fetch-patched override"
```

---

## Task 21: Empty Go module wiring

**Files:**
- Create: `internal/modules/templates/module.go`
- Create: `internal/modules/editor_sessions/module.go`
- Create: `internal/modules/document_revisions/module.go`
- Modify: `apps/api/cmd/metaldocs-api/main.go`

- [ ] **Step 1: Write placeholder module**

`internal/modules/templates/module.go`:
```go
// Package templates will own template CRUD + publish in W2.
// W1 scaffolds the package so import graph compiles.
package templates

// Module is a placeholder; real wiring lands in W2 plan.
type Module struct{}

func New() *Module { return &Module{} }
```

- [ ] **Step 2: Write placeholder editor_sessions**

`internal/modules/editor_sessions/module.go`:
```go
// Package editor_sessions will own pessimistic editor locks in W3.
package editor_sessions

type Module struct{}

func New() *Module { return &Module{} }
```

- [ ] **Step 3: Write placeholder document_revisions**

`internal/modules/document_revisions/module.go`:
```go
// Package document_revisions will own content-addressed revision CAS in W3.
package document_revisions

type Module struct{}

func New() *Module { return &Module{} }
```

- [ ] **Step 4: Read main.go**

```bash
rtk read apps/api/cmd/metaldocs-api/main.go
```

Identify the imports block.

- [ ] **Step 5: Add no-op imports to main.go**

Near existing module imports, add:
```go
_ "metaldocs/internal/modules/templates"
_ "metaldocs/internal/modules/editor_sessions"
_ "metaldocs/internal/modules/document_revisions"
```

(Blank imports force compile; no runtime behavior.)

- [ ] **Step 6: Build API binary**

```bash
go build ./apps/api/cmd/metaldocs-api
```

Expected: zero errors; binary produced (or `-o /dev/null` — adapt to CI convention).

- [ ] **Step 7: Commit**

```bash
rtk git add internal/modules/templates internal/modules/editor_sessions internal/modules/document_revisions apps/api/cmd/metaldocs-api/main.go
rtk git commit -m "feat(docx-v2): placeholder Go modules (templates, sessions, revisions)"
```

---

## Task 22: `docgen_v2` Go client (health-only)

**Files:**
- Create: `internal/platform/servicebus/docgen_v2_client.go`
- Create: `internal/platform/servicebus/docgen_v2_client_test.go`

- [ ] **Step 1: Write failing test using httptest**

`internal/platform/servicebus/docgen_v2_client_test.go`:
```go
package servicebus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/platform/servicebus"
)

func TestDocgenV2Client_Health_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Service-Token"); got != "" {
			t.Fatalf("/health must NOT require token; got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"test"}`))
	}))
	defer srv.Close()

	c := servicebus.NewDocgenV2Client(srv.URL, "shh", 2*time.Second)
	ver, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "test" {
		t.Fatalf("expected version 'test', got %q", ver)
	}
}

func TestDocgenV2Client_Health_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := servicebus.NewDocgenV2Client(srv.URL, "shh", 500*time.Millisecond)
	if _, err := c.Health(context.Background()); err == nil {
		t.Fatal("expected error on 502")
	}
}
```

- [ ] **Step 2: Run test (FAIL — package absent)**

```bash
go test ./internal/platform/servicebus/...
```

- [ ] **Step 3: Write implementation**

`internal/platform/servicebus/docgen_v2_client.go`:
```go
// Package servicebus holds internal service-to-service clients.
// docgen_v2_client is the minimal W1 client (health only).
package servicebus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DocgenV2Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewDocgenV2Client(baseURL, token string, timeout time.Duration) *DocgenV2Client {
	return &DocgenV2Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health pings /health (no auth). Returns remote version string on 200.
func (c *DocgenV2Client) Health(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docgen-v2 health: unexpected status %d", resp.StatusCode)
	}
	var out healthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("docgen-v2 health: decode: %w", err)
	}
	if out.Status != "ok" {
		return "", fmt.Errorf("docgen-v2 health: status=%q", out.Status)
	}
	return out.Version, nil
}
```

- [ ] **Step 4: Run test (PASS)**

```bash
go test ./internal/platform/servicebus/... -v
```

Expected: both subtests PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/platform/servicebus
rtk git commit -m "feat(docx-v2): Go client for docgen-v2 /health"
```

---

## Task 23: MinIO bucket seeding script

**Files:**
- Create: `scripts/docx-v2-seed-minio.sh`

- [ ] **Step 1: Write script**

```bash
#!/usr/bin/env bash
set -euo pipefail

# Seeds MinIO with the docx-v2 tenants bucket + verifies access.
# Uses the mc client already vendored in the minio/mc image via docker run.

MINIO_HOST="${MINIO_HOST:-http://minio:9000}"
MINIO_ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_ROOT_PASSWORD:-minioadmin}"
BUCKET="${DOCX_V2_BUCKET:-metaldocs-docx-v2}"
NETWORK="${COMPOSE_NETWORK:-metaldocs_default}"

docker run --rm --network "$NETWORK" \
  -e MC_HOST_local="http://${MINIO_ACCESS_KEY}:${MINIO_SECRET_KEY}@minio:9000" \
  minio/mc:RELEASE.2024-04-18T16-45-29Z \
  sh -c "
    mc mb -p local/${BUCKET} || true
    mc anonymous set none local/${BUCKET}
    mc ls local/
  "

echo "OK: bucket ${BUCKET} ready"
```

- [ ] **Step 2: Make executable + run**

```bash
chmod +x scripts/docx-v2-seed-minio.sh
bash scripts/docx-v2-seed-minio.sh
```

Expected: final line `OK: bucket metaldocs-docx-v2 ready`. `mc ls` shows the bucket.

- [ ] **Step 3: Commit**

```bash
rtk git add scripts/docx-v2-seed-minio.sh
rtk git commit -m "chore(docx-v2): MinIO seeding script"
```

---

## Task 24: CI — docx-v2 lint + typecheck + test job

**Files:**
- Create: `.github/workflows/docx-v2-ci.yml`

- [ ] **Step 1: Write workflow**

```yaml
name: docx-v2 CI

on:
  pull_request:
    paths:
      - 'apps/docgen-v2/**'
      - 'packages/**'
      - 'migrations/0101_*'
      - 'migrations/0102_*'
      - 'migrations/0103_*'
      - 'migrations/0104_*'
      - 'migrations/0105_*'
      - 'migrations/0106_*'
      - 'migrations/0107_*'
      - 'migrations/0108_*'
      - 'internal/platform/config/feature_flags*.go'
      - 'internal/platform/featureflags/**'
      - 'internal/platform/servicebus/docgen_v2_client*.go'
      - 'internal/modules/templates/**'
      - 'internal/modules/editor_sessions/**'
      - 'internal/modules/document_revisions/**'
      - 'frontend/apps/web/src/features/featureFlags.ts'
      - 'frontend/apps/web/src/features/__tests__/featureFlags.docxV2.test.ts'
      - 'tests/docx_v2/**'
      - 'scripts/docx-v2-*'
      - '.github/workflows/docx-v2-ci.yml'

jobs:
  node:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20.11.0, cache: npm }
      - run: npm ci --include-workspace-root
      - run: npm run typecheck:docx-v2
      - run: npm run test:docx-v2
      - run: npm run build:docx-v2
  frontend-docxv2-flag:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 20.11.0, cache: npm, cache-dependency-path: frontend/apps/web/package-lock.json }
      - run: npm ci
        working-directory: frontend/apps/web
      - run: npm run test -- featureFlags.docxV2.test.ts
        working-directory: frontend/apps/web
  go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 1.25.x, cache: true }
      - run: go build ./...
      - run: go test ./internal/platform/config/... ./internal/platform/featureflags/... ./internal/platform/servicebus/... ./internal/modules/templates/... ./internal/modules/editor_sessions/... ./internal/modules/document_revisions/... ./tests/docx_v2/...
  migration-smoke:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: metaldocs
          POSTGRES_PASSWORD: metaldocs
          POSTGRES_DB: metaldocs
        options: >-
          --health-cmd "pg_isready -U metaldocs"
          --health-interval 5s
          --health-timeout 3s
          --health-retries 10
        ports: [ "5432:5432" ]
    steps:
      - uses: actions/checkout@v4
      - name: Apply docx-v2 migrations
        env:
          PGPASSWORD: metaldocs
        run: |
          for f in migrations/0101_*.sql migrations/0102_*.sql migrations/0103_*.sql \
                   migrations/0104_*.sql migrations/0105_*.sql migrations/0106_*.sql \
                   migrations/0107_*.sql migrations/0108_*.sql; do
            echo "==> $f"
            psql -h 127.0.0.1 -U metaldocs -d metaldocs -v ON_ERROR_STOP=1 -f "$f"
          done
      - name: Verify tables + indexes
        env:
          PGPASSWORD: metaldocs
          SERVICE: 127.0.0.1
        run: |
          for t in templates template_versions documents_v2 editor_sessions document_revisions autosave_pending_uploads document_checkpoints template_audit_log; do
            psql -h 127.0.0.1 -U metaldocs -d metaldocs -tAc "SELECT to_regclass('public.$t')" | grep -qx "$t"
          done
          for idx in idx_one_draft_per_template idx_one_active_session_per_doc idx_pending_expired idx_documents_v2_form_data_gin; do
            psql -h 127.0.0.1 -U metaldocs -d metaldocs -tAc "SELECT 1 FROM pg_indexes WHERE indexname='$idx'" | grep -qx 1
          done
```

- [ ] **Step 2: Lint workflow YAML locally**

```bash
python -c "import yaml, sys; yaml.safe_load(open('.github/workflows/docx-v2-ci.yml')); print('ok')"
```

Expected: `ok`.

- [ ] **Step 3: Commit**

```bash
rtk git add .github/workflows/docx-v2-ci.yml
rtk git commit -m "ci(docx-v2): docx-v2 CI pipeline (node, go, migration smoke)"
```

---

## Task 25: Governance compliance (runbook + tests stub + isolation check)

**Context:** `scripts/check-governance.ps1` enforces three rules that this plan would otherwise trip:

1. Any change under `internal/modules/**` requires a change under `tests/**`.
2. Any change under `deploy/**` or `scripts/**` requires a change under `docs/runbooks/**`.
3. Any `internal/modules/**/delivery/http/**.go` change requires an `api/openapi/v1/openapi.yaml` change (not triggered — we do not add HTTP handlers in W1).

This task creates the minimum genuine artifacts to satisfy rules 1 and 2 for the whole W1 PR, AND adds a docx-v2 isolation check forbidding CK5 path edits.

**Files:**
- Create: `docs/runbooks/docx-v2-w1-scaffold.md`
- Create: `tests/docx_v2/scaffold_smoke_test.go`
- Modify: `.github/workflows/governance-check.yml`

- [ ] **Step 1: Read current governance check**

```bash
rtk read scripts/check-governance.ps1
rtk read .github/workflows/governance-check.yml
```

- [ ] **Step 2: Write runbook**

`docs/runbooks/docx-v2-w1-scaffold.md`:
```markdown
# Runbook — docx-v2 W1 scaffold

This runbook covers the greenfield docx-editor platform scaffold introduced
in W1 (see `docs/superpowers/plans/2026-04-18-docx-editor-w1-scaffold.md`).
It is referenced by `scripts/check-governance.ps1` which requires a runbook
entry for any `deploy/` or `scripts/` change.

## What W1 adds

- New Fastify service `apps/docgen-v2` exposing only `/health` (port 3100).
- New Postgres tables 0101–0108 (templates, template_versions, documents_v2,
  editor_sessions, document_revisions, autosave_pending_uploads,
  document_checkpoints, template_audit_log).
- New `METALDOCS_DOCX_V2_ENABLED` feature flag wired through Go config and
  the `GET /api/v1/feature-flags` response.
- Empty npm workspace packages under `packages/*` (business logic arrives
  in W2–W4 plans).

## Operator bring-up

```
export DOCGEN_V2_SERVICE_TOKEN=$(openssl rand -hex 24)
docker compose -f deploy/compose/docker-compose.yml \
  up -d postgres minio gotenberg docgen-v2
bash scripts/docx-v2-verify-migrations.sh
bash scripts/docx-v2-seed-minio.sh
curl -f http://127.0.0.1:3100/health
```

## Rollback

W1 is pure-additive: no existing table altered, no existing service
modified. Rollback = drop the 8 new tables in reverse FK order:

```
DROP TABLE template_audit_log, document_checkpoints, autosave_pending_uploads,
           document_revisions, editor_sessions, documents_v2,
           template_versions, templates;
```

…and remove `docgen-v2` from the compose file. No data loss beyond new-path
state.

## Known limits (carried to W2)

- Per-tenant flag resolution not implemented; `METALDOCS_DOCX_V2_ENABLED`
  is global-only.
- `/render/docx` and other docgen-v2 routes return 404 by design.
- OOXML parser / validators arrive in W2.
```

- [ ] **Step 3: Write tests stub (real compile-testable smoke)**

`tests/docx_v2/scaffold_smoke_test.go`:
```go
// Package docx_v2_test asserts the W1 scaffold compiles and is wired to
// the same import graph as the main API. This file intentionally depends
// only on the three placeholder modules so governance-check's "internal/modules
// change needs tests/" rule is satisfied, and so a future compile break in
// those placeholders is caught at CI time.
package docx_v2_test

import (
	"testing"

	"metaldocs/internal/modules/document_revisions"
	"metaldocs/internal/modules/editor_sessions"
	"metaldocs/internal/modules/templates"
)

func TestScaffoldCompiles(t *testing.T) {
	if templates.New() == nil {
		t.Fatal("templates.New() returned nil")
	}
	if editor_sessions.New() == nil {
		t.Fatal("editor_sessions.New() returned nil")
	}
	if document_revisions.New() == nil {
		t.Fatal("document_revisions.New() returned nil")
	}
}
```

- [ ] **Step 4: Run runbook governance dry-run locally**

```bash
powershell -ExecutionPolicy Bypass -File scripts/check-governance.ps1 -BaseRef main
```

Expected: `[governance-check] OK`.

- [ ] **Step 5: Run the new test**

```bash
go test ./tests/docx_v2/...
```

Expected: PASS.

- [ ] **Step 6: Add `docx-v2-isolation` job to the workflow**

Edit `.github/workflows/governance-check.yml`. Append new job at end of `jobs:` block (YAML indent: 2 spaces):

```yaml
  docx-v2-isolation:
    name: docx-v2 PR must not touch CK5 paths
    runs-on: ubuntu-latest
    if: contains(github.event.pull_request.title, 'docx-v2') || startsWith(github.event.pull_request.head.ref, 'feat/docx-v2-')
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - name: Fail if PR edits CK5 paths
        run: |
          base="${{ github.event.pull_request.base.sha }}"
          head="${{ github.event.pull_request.head.sha }}"
          bad=$(git diff --name-only "$base" "$head" | grep -E '^(frontend/apps/web/src/features/documents/ck5/|apps/ck5-export/|apps/ck5-studio/|apps/docgen/|internal/modules/documents/application/service_ck5_|internal/modules/documents/delivery/http/handler_ck5_)' || true)
          if [[ -n "$bad" ]]; then
            echo "::error::docx-v2 PR must not touch CK5 paths:"
            echo "$bad"
            exit 1
          fi
          echo "OK: no CK5 paths touched"
```

- [ ] **Step 7: Validate YAML**

```bash
python -c "import yaml; yaml.safe_load(open('.github/workflows/governance-check.yml')); print('ok')"
```

Expected: `ok`.

- [ ] **Step 8: Commit**

```bash
rtk git add docs/runbooks/docx-v2-w1-scaffold.md tests/docx_v2/scaffold_smoke_test.go .github/workflows/governance-check.yml
rtk git commit -m "ci(docx-v2): governance compliance (runbook, tests stub, CK5 isolation)"
```

---

## Task 26: End-to-end scaffold smoke

**Files:** none (script-only verification).

- [ ] **Step 1: Full local bring-up**

```bash
export DOCGEN_V2_SERVICE_TOKEN=0123456789abcdef
docker compose -f deploy/compose/docker-compose.yml up -d postgres minio gotenberg docgen-v2
sleep 8
```

- [ ] **Step 2: Apply migrations**

```bash
for f in migrations/0101_*.sql migrations/0102_*.sql migrations/0103_*.sql \
         migrations/0104_*.sql migrations/0105_*.sql migrations/0106_*.sql \
         migrations/0107_*.sql migrations/0108_*.sql; do
  docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
    psql -U ${PGUSER:-metaldocs} -d ${PGDATABASE:-metaldocs} -v ON_ERROR_STOP=1 -f "/migrations/$(basename $f)" || exit 1
done
```

- [ ] **Step 3: Verify tables**

```bash
bash scripts/docx-v2-verify-migrations.sh
```

Expected: `OK: all 8 tables + 4 critical indexes present`.

- [ ] **Step 4: Seed MinIO**

```bash
bash scripts/docx-v2-seed-minio.sh
```

Expected: `OK: bucket metaldocs-docx-v2 ready`.

- [ ] **Step 5: Health-check docgen-v2**

```bash
curl -s http://127.0.0.1:3100/health
```

Expected: `{"status":"ok","version":"dev"}`.

- [ ] **Step 6: Protected path rejects without token**

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X POST http://127.0.0.1:3100/render/docx
```

Expected: `401`.

- [ ] **Step 7: Protected path with token returns 404 (route not registered yet) — confirm auth middleware passes**

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X POST -H "X-Service-Token: $DOCGEN_V2_SERVICE_TOKEN" http://127.0.0.1:3100/render/docx
```

Expected: `404` (Fastify default for unhandled route). Auth passed; route implementation arrives in Plan B.

- [ ] **Step 8: Go build of API succeeds with new placeholder module imports**

```bash
go build ./apps/api/cmd/metaldocs-api
```

Expected: zero errors.

- [ ] **Step 9: Full Go test suite for new packages**

```bash
go test ./internal/platform/config/... \
        ./internal/platform/featureflags/... \
        ./internal/platform/servicebus/... \
        ./tests/docx_v2/... -v
```

Expected: all PASS.

- [ ] **Step 10: Full Node test suite across workspaces**

```bash
npm run test:docx-v2
```

Expected: every workspace reports `Tests N passed` or `-- no tests` (docx-editor fork).

- [ ] **Step 11: Tear down**

```bash
docker compose -f deploy/compose/docker-compose.yml stop docgen-v2
```

- [ ] **Step 12: No commit (verification only).**

If any step fails, return to the task that created the artifact, fix, and re-run this task.

---

## Spec Coverage Checklist

Maps each spec §Components / §Architecture item to a task in this plan.

| Spec item | Task |
|-|-|
| §Architecture → service topology (Go API + docgen-v2 + Postgres + MinIO + Gotenberg) | 8, 22, 26 |
| §Architecture → monorepo layout (`apps/docgen-v2`, `packages/*`) | 1–7 |
| §Components → `templates` table | 10 |
| §Components → `template_versions` table + draft-unique index | 11 |
| §Components → `documents` table (named `documents_v2` pre-W5) | 12 |
| §Components → `editor_sessions` table + active-unique index | 13 |
| §Components → `document_revisions` table + FK to self + hash unique | 14 |
| §Components → `autosave_pending_uploads` table + composite unique | 15 |
| §Components → `document_checkpoints` table | 16 |
| §Components → `template_audit_log` append-only | 17 |
| §Components → S3 key scheme + bucket policy (bucket created, policy in W2) | 23 |
| §Components → docgen-v2 HTTP surface (only `/health` this week; rest in W2–W4) | 7, 8, 22 |
| §Components → `packages/shared-tokens` (empty barrel; parser in W2) | 3 |
| §Components → `packages/editor-ui` (empty barrel; wrapper in W2) | 4 |
| §Components → `packages/form-ui` (empty barrel; renderer in W2) | 5 |
| §Components → `packages/shared-types` (empty barrel) | 2 |
| §Components → `packages/docx-editor` subtree scaffold | 6 |
| §Architecture → feature flag `feature.docx_v2_enabled` (Go + frontend) | 19, 20 |
| §Rollout → W1 "scaffold new packages + modules alongside old code. New DB tables introduced; no drops." | all |
| §Rollout → "Feature flag METALDOCS_DOCX_V2_ENABLED (per-tenant, default OFF)" — per-tenant is W4 scope, W1 is global | 19 |
| §Error handling → audit log append-only grants | 17 |
| §Testing → testcontainers note | deferred; W1 uses docker-compose postgres + CI service container |

No gaps in W1 scope. W2+ items carried forward as planned.

---

## Out of Scope (W1)

- Any business logic (template/document CRUD, publish, autosave, render) → W2–W4.
- Per-tenant flag resolution → W4.
- Any UI under `frontend/apps/web/src/features/templates` or `features/documents` v2 → W2–W3.
- `parseDocxTokens` parser → W2.
- OOXML whitelist enforcement → W2.
- Gotenberg integration code → W4.
- Destructive CK5 deletions → W5.
- Testcontainers migration from in-memory fakes → not in scope (follow existing pattern).

---
