# Docgen Minimal Test Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal `apps/docgen` harness that typechecks, starts a Node server, and returns a valid `.docx` on a single `/generate` request.

**Architecture:** A tiny Express service in `apps/docgen` uses `docx` to produce an in-memory `.docx` buffer. A PowerShell harness script runs `tsc --noEmit`, starts the server, and curls `/generate` with a sample payload.

**Tech Stack:** Node.js, TypeScript, Express, `docx` (npm).

---

## File Structure

Create:
- `apps/docgen/package.json` — dependencies and scripts
- `apps/docgen/tsconfig.json` — typecheck config
- `apps/docgen/tsconfig.build.json` — emit to `dist`
- `apps/docgen/src/index.ts` — Express server + `/generate`
- `apps/docgen/src/generate.ts` — minimal docx generator
- `apps/docgen/scripts/sample-payload.json` — minimal payload for curl
- `apps/docgen/scripts/harness.ps1` — runs typecheck, build, server, curl

Modify (if needed by repo conventions later, not in this harness):
- none

---

### Task 1: Add the Harness Script and Sample Payload (Failing First)

**Files:**
- Create: `apps/docgen/scripts/sample-payload.json`
- Create: `apps/docgen/scripts/harness.ps1`

- [ ] **Step 1: Create sample payload**

```json
{
  "documentType": "PO",
  "documentCode": "PO-01",
  "title": "Procedimento Operacional",
  "sections": {}
}
```

- [ ] **Step 2: Create harness script**

```powershell
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

try {
  Write-Host "==> Typecheck (tsc --noEmit)"
  npx tsc --noEmit

  Write-Host "==> Build (tsc -p tsconfig.build.json)"
  npx tsc -p tsconfig.build.json

  Write-Host "==> Start server (node dist/index.js)"
  $proc = Start-Process -FilePath "node" -ArgumentList "dist/index.js" -PassThru -NoNewWindow
  Start-Sleep -Seconds 2

  Write-Host "==> POST /generate"
  $resp = curl.exe -s -D - -o "$env:TEMP\\docgen-harness.docx" `
    -H "Content-Type: application/json" `
    -X POST "http://localhost:3001/generate" `
    --data-binary "@$PSScriptRoot\\sample-payload.json"

  $len = (Get-Item "$env:TEMP\\docgen-harness.docx").Length
  if ($len -le 0) { throw "DOCX is empty" }

  if ($resp -notmatch "application/vnd.openxmlformats-officedocument.wordprocessingml.document") {
    throw "Unexpected content type"
  }

  Write-Host "OK: DOCX size = $len bytes"
}
finally {
  if ($proc -and !$proc.HasExited) { Stop-Process -Id $proc.Id }
  Pop-Location
}
```

- [ ] **Step 3: Run harness script to confirm failure (expected)**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`  
Expected: FAIL because `apps/docgen` does not yet have `package.json`/`tsconfig`/`dist`.

---

### Task 2: Scaffold `apps/docgen` Package and TypeScript Config

**Files:**
- Create: `apps/docgen/package.json`
- Create: `apps/docgen/tsconfig.json`
- Create: `apps/docgen/tsconfig.build.json`

- [ ] **Step 1: Create `package.json`**

```json
{
  "name": "@metaldocs/docgen",
  "private": true,
  "type": "module",
  "scripts": {
    "typecheck": "tsc --noEmit",
    "build": "tsc -p tsconfig.build.json",
    "start": "node dist/index.js"
  },
  "dependencies": {
    "docx": "^9.0.0",
    "express": "^4.19.2"
  },
  "devDependencies": {
    "@types/express": "^4.17.21",
    "typescript": "^5.4.5"
  }
}
```

- [ ] **Step 2: Create `tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "skipLibCheck": true,
    "noEmit": true,
    "rootDir": "src"
  },
  "include": ["src"]
}
```

- [ ] **Step 3: Create `tsconfig.build.json`**

```json
{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "noEmit": false,
    "outDir": "dist"
  }
}
```

- [ ] **Step 4: Install dependencies**

Run: `cd apps/docgen; npm install`  
Expected: SUCCESS

---

### Task 3: Implement Minimal Docgen Server and Generator

**Files:**
- Create: `apps/docgen/src/generate.ts`
- Create: `apps/docgen/src/index.ts`

- [ ] **Step 1: Implement `generate.ts`**

```ts
import { Document, Packer, Paragraph, TextRun } from "docx";

export async function generateDocx(_: unknown): Promise<Uint8Array> {
  const doc = new Document({
    sections: [
      {
        children: [
          new Paragraph({
            children: [new TextRun({ text: "MetalDocs Docgen Harness", bold: true })],
          }),
          new Paragraph("Document generated for harness validation.")
        ],
      },
    ],
  });

  return Packer.toBuffer(doc);
}
```

- [ ] **Step 2: Implement `index.ts`**

```ts
import express from "express";
import { generateDocx } from "./generate.js";

const app = express();
app.use(express.json({ limit: "10mb" }));

app.post("/generate", async (req, res) => {
  try {
    const buf = await generateDocx(req.body);
    res.setHeader(
      "Content-Type",
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    );
    res.send(Buffer.from(buf));
  } catch (err) {
    res.status(500).json({ error: "DOCGEN_GENERATE_FAILED" });
  }
});

app.listen(3001, () => {
  console.log("docgen listening on :3001");
});
```

- [ ] **Step 3: Typecheck**

Run: `cd apps/docgen; npx tsc --noEmit`  
Expected: PASS

- [ ] **Step 4: Build**

Run: `cd apps/docgen; npx tsc -p tsconfig.build.json`  
Expected: `dist/` created with `index.js` and `generate.js`

- [ ] **Step 5: Run harness script (pass criteria)**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`  
Expected: PASS; non-zero `.docx` bytes and correct `Content-Type`

---

## Plan Self-Review

- [x] Spec coverage: this plan addresses the minimal docgen test harness requirements.
- [x] Placeholder scan: no TBDs or implied steps.
- [x] Type consistency: file names and commands are consistent across tasks.

---

**Plan complete and saved to `docs/superpowers/plans/2026-03-31-docgen-minimal-harness.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
