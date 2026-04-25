import type { Placeholder } from '../templates/placeholder-types';
import { getPlaceholderValues, getZoneContents } from './v2/api/documentsV2';
import type { PlaceholderValueDTO, ZoneContentDTO } from './v2/api/documentsV2';

export interface FillInData {
  bodyDocx: Uint8Array;
  placeholderValues: PlaceholderValueDTO[];
  zoneContents: ZoneContentDTO[];
  placeholderSchema: Placeholder[];
}

interface WirePlaceholder { id: string; label: string; type: string; required: boolean; options?: string[]; max_length?: number; resolver_key?: string; }

interface FillInSchemaResponse {
  data: {
    placeholder_schema: WirePlaceholder[];
  };
}

function placeholderFromWire(w: WirePlaceholder): Placeholder {
  return {
    id: w.id,
    label: w.label,
    type: w.type as Placeholder['type'],
    ...(w.required ? { required: true } : {}),
    ...(w.options ? { options: w.options } : {}),
    ...(w.max_length != null ? { maxLength: w.max_length } : {}),
    ...(w.resolver_key != null ? { resolverKey: w.resolver_key } : {}),
  };
}

export async function loadFillInData(docId: string): Promise<FillInData> {
  const [schema, values, zones] = await Promise.all([
    fetch(`/api/v2/documents/${docId}/fill-in-schema`).then((r) => {
      if (!r.ok) throw Object.assign(new Error(`http_${r.status}`), { status: r.status });
      return r.json() as Promise<FillInSchemaResponse>;
    }),
    getPlaceholderValues(docId),
    getZoneContents(docId),
  ]);

  return {
    bodyDocx: new Uint8Array(),
    placeholderValues: values,
    zoneContents: zones,
    placeholderSchema: (schema.data.placeholder_schema ?? []).map(placeholderFromWire),
  };
}
