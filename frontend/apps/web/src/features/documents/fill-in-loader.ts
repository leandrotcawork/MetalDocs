import type { Placeholder, EditableZone } from '../templates/placeholder-types';
import { getPlaceholderValues, getZoneContents } from './v2/api/documentsV2';
import type { PlaceholderValueDTO, ZoneContentDTO } from './v2/api/documentsV2';

export interface FillInData {
  bodyDocx: Uint8Array;
  placeholderValues: PlaceholderValueDTO[];
  zoneContents: ZoneContentDTO[];
  placeholderSchema: Placeholder[];
  zoneSchema: EditableZone[];
}

interface DocumentDetailResponse {
  data: {
    body_url: string;
    placeholder_schema: Placeholder[];
    zone_schema: EditableZone[];
  };
}

export async function loadFillInData(docId: string): Promise<FillInData> {
  const [detail, values, zones] = await Promise.all([
    fetch(`/api/v2/documents/${docId}`).then((r) => {
      if (!r.ok) throw Object.assign(new Error(`http_${r.status}`), { status: r.status });
      return r.json() as Promise<DocumentDetailResponse>;
    }),
    getPlaceholderValues(docId),
    getZoneContents(docId),
  ]);

  const bodyBuf = await fetch(detail.data.body_url).then((r) => {
    if (!r.ok) throw Object.assign(new Error(`http_${r.status}`), { status: r.status });
    return r.arrayBuffer();
  });

  return {
    bodyDocx: new Uint8Array(bodyBuf),
    placeholderValues: values,
    zoneContents: zones,
    placeholderSchema: detail.data.placeholder_schema ?? [],
    zoneSchema: detail.data.zone_schema ?? [],
  };
}
