import { createHash } from 'node:crypto';

export interface PdfJobInput {
  tenant_id: string;
  revision_id: string;
  final_docx_s3_key: string;
}

export interface PdfJobResult {
  final_pdf_s3_key: string;
  pdf_hash: string;
  pdf_generated_at: string;
}

export interface PdfJobDeps {
  gotenbergUrl: string;
  getObject: (key: string) => Promise<Buffer>;
  putObject: (key: string, data: Buffer, contentType: string) => Promise<void>;
  sleep?: (ms: number) => Promise<void>;
  now?: () => Date;
}

const MAX_ATTEMPTS = 3;
const BASE_BACKOFF_MS = 250;
const PDF_CONTENT_TYPE = 'application/pdf';
const DOCX_CONTENT_TYPE =
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document';

export async function runPdfJob(
  input: PdfJobInput,
  deps: PdfJobDeps,
): Promise<PdfJobResult> {
  const docxBuffer = await deps.getObject(input.final_docx_s3_key);
  const pdfBuffer = await convertWithRetry(docxBuffer, deps);

  const pdfKey = `${input.final_docx_s3_key}.pdf`;
  await deps.putObject(pdfKey, pdfBuffer, PDF_CONTENT_TYPE);

  const pdfHash = createHash('sha256').update(pdfBuffer).digest('hex');
  const generatedAt = (deps.now ? deps.now() : new Date()).toISOString();

  return {
    final_pdf_s3_key: pdfKey,
    pdf_hash: pdfHash,
    pdf_generated_at: generatedAt,
  };
}

async function convertWithRetry(
  docx: Buffer,
  deps: PdfJobDeps,
): Promise<Buffer> {
  const sleep = deps.sleep ?? defaultSleep;
  const url = `${deps.gotenbergUrl.replace(/\/+$/, '')}/forms/libreoffice/convert`;

  let lastErr: Error | null = null;
  for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
    const form = new FormData();
    form.append(
      'files',
      new Blob([docx], { type: DOCX_CONTENT_TYPE }),
      'document.docx',
    );
    const res = await fetch(url, { method: 'POST', body: form });
    if (res.ok) {
      return Buffer.from(await res.arrayBuffer());
    }
    lastErr = new Error(`gotenberg status ${res.status}`);
    if (res.status < 500) break;
    if (attempt < MAX_ATTEMPTS) {
      await sleep(BASE_BACKOFF_MS * 2 ** (attempt - 1));
    }
  }
  throw lastErr ?? new Error('gotenberg: unknown failure');
}

async function defaultSleep(ms: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, ms));
}
