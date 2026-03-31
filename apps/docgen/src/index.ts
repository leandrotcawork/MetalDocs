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
  } catch {
    res.status(500).json({ error: "DOCGEN_GENERATE_FAILED" });
  }
});

const port = Number(process.env.PORT ?? 3001);

app.listen(port, () => {
  console.log(`docgen listening on :${port}`);
});
