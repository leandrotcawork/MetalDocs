import express from "express";
import { generateBrowserDocx } from "./generateBrowser.js";
import { generateDocx } from "./generate.js";
import { exportMDDMToDocx } from "./mddm/exporter.js";

const app = express();
app.use(express.json({ limit: "10mb" }));

app.get("/", (_req, res) => {
  res.status(200).json({ ok: true });
});

app.post("/generate", async (req, res) => {
  try {
    const buf = await generateDocx(req.body);
    res.setHeader(
      "Content-Type",
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    );
    res.send(Buffer.from(buf));
  } catch (err) {
    const message = err instanceof Error ? err.message : "DOCGEN_GENERATE_FAILED";
    if (message.startsWith("DOCGEN_INVALID_")) {
      res.status(400).json({ error: message });
      return;
    }
    console.error("DOCGEN_GENERATE_FAILED", err);
    res.status(500).json({ error: "DOCGEN_GENERATE_FAILED" });
  }
});

app.post("/generate-browser", async (req, res) => {
  try {
    const buf = await generateBrowserDocx(req.body);
    res.setHeader(
      "Content-Type",
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    );
    res.send(Buffer.from(buf));
  } catch (err) {
    const message = err instanceof Error ? err.message : "DOCGEN_GENERATE_FAILED";
    if (message.startsWith("DOCGEN_INVALID_")) {
      res.status(400).json({ error: message });
      return;
    }
    console.error("DOCGEN_GENERATE_BROWSER_FAILED", err);
    res.status(500).json({ error: "DOCGEN_GENERATE_FAILED" });
  }
});

app.post("/render/mddm-docx", express.json({ limit: "10mb" }), async (req, res) => {
  try {
    const buf = await exportMDDMToDocx(req.body);
    res.setHeader(
      "Content-Type",
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    );
    res.send(Buffer.from(buf));
  } catch (err: any) {
    res.status(500).json({ error: "render_failed", message: err.message });
  }
});

const port = Number(process.env.PORT ?? 3001);

app.listen(port, () => {
  console.log(`docgen listening on :${port}`);
});
