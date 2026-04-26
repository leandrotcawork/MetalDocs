import { createHash } from 'node:crypto';
import { processTemplateDetailed } from '@eigenpal/docx-js-editor/headless';

export interface ProcessDocxResult {
  buffer: Uint8Array;
  contentHash: string;
  unreplacedVars: string[];
}

export async function processDocx(
  templateBuffer: Uint8Array,
  formData: Record<string, unknown>,
): Promise<ProcessDocxResult> {
  // processTemplateDetailed expects Record<string, string> — coerce values to string
  const variables: Record<string, string> = {};
  for (const [k, v] of Object.entries(formData)) {
    variables[k] = v == null ? '' : String(v);
  }

  const result = processTemplateDetailed(
    templateBuffer.buffer.slice(
      templateBuffer.byteOffset,
      templateBuffer.byteOffset + templateBuffer.byteLength,
    ) as ArrayBuffer,
    variables,
    { nullGetter: 'empty' },
  );

  const buf = new Uint8Array(result.buffer);
  const contentHash = createHash('sha256').update(buf).digest('hex');

  return {
    buffer: buf,
    contentHash,
    unreplacedVars: result.unreplacedVariables ?? [],
  };
}
