import express from "express";
import { generateDocx } from "./generate.js";

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

const port = Number(process.env.PORT ?? 3001);

app.listen(port, () => {
  console.log(`docgen listening on :${port}`);
});
