import { createHash } from 'node:crypto';
import { processTemplateDetailed } from '@eigenpal/docx-js-editor';
import { injectZones } from './zoneInjection.js';
import { SubBlockRegistry } from './subblocks/registry.js';
import { registerV1Builtins } from './subblocks/builtins.js';

export interface FanoutInput {
  bodyDocx: Uint8Array;
  placeholderValues: Record<string, string>;
  zoneContent: Record<string, string>;
  compositionConfig: {
    header_sub_blocks: string[];
    footer_sub_blocks: string[];
    sub_block_params: Record<string, Record<string, unknown>>;
  };
  resolvedValues: Record<string, unknown>;
}

export interface FanoutResult {
  buffer: Uint8Array;
  contentHash: string;
  unreplacedVars: string[];
}

export async function fanout(input: FanoutInput): Promise<FanoutResult> {
  const subReg = new SubBlockRegistry();
  registerV1Builtins(subReg);

  const headerOoxml = (
    await Promise.all(
      input.compositionConfig.header_sub_blocks.map((k) =>
        subReg.render(k, {
          params: input.compositionConfig.sub_block_params[k] ?? {},
          values: input.resolvedValues,
        }),
      ),
    )
  ).join('');
  const footerOoxml = (
    await Promise.all(
      input.compositionConfig.footer_sub_blocks.map((k) =>
        subReg.render(k, {
          params: input.compositionConfig.sub_block_params[k] ?? {},
          values: input.resolvedValues,
        }),
      ),
    )
  ).join('');

  const withZones = await injectZones(input.bodyDocx, input.zoneContent);

  const variables: Record<string, string> = {
    ...input.placeholderValues,
    __header_composition__: headerOoxml,
    __footer_composition__: footerOoxml,
  };

  const result = processTemplateDetailed(
    withZones.buffer.slice(
      withZones.byteOffset,
      withZones.byteOffset + withZones.byteLength,
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
