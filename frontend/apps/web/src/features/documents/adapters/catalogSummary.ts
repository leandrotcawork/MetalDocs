import type { DocumentProfileItem, ProcessAreaItem, SearchDocumentItem } from "../../../lib.types";

export type ProfileAccordionSummary = {
  code: string;
  label: string;
  count: number;
  areas: Array<{ label: string; count: number }>;
};

type DocumentAggregate = {
  profileCountByCode: Record<string, number>;
  profileAreaCountByCode: Record<string, Record<string, number>>;
};

function normalizeAreaLabel(value: string): string {
  const trimmed = value.trim();
  if (trimmed === "") {
    return "Sem area";
  }
  return trimmed;
}

function areaLabelByCode(processAreas: ProcessAreaItem[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const area of processAreas) {
    if (!area.code) {
      continue;
    }
    const key = area.code.trim().toLowerCase();
    if (key === "") {
      continue;
    }
    out[key] = normalizeAreaLabel(area.name || area.code);
  }
  return out;
}

function aggregateDocuments(documents: SearchDocumentItem[], processAreas: ProcessAreaItem[]): DocumentAggregate {
  const profileCountByCode: Record<string, number> = {};
  const profileAreaCountByCode: Record<string, Record<string, number>> = {};
  const labelsByAreaCode = areaLabelByCode(processAreas);

  for (const document of documents) {
    const profileCode = document.documentProfile;
    if (!profileCode) {
      continue;
    }

    profileCountByCode[profileCode] = (profileCountByCode[profileCode] ?? 0) + 1;

    const areaCode = (document.processArea ?? "").trim().toLowerCase();
    const areaLabel = areaCode === "" ? "Sem area" : (labelsByAreaCode[areaCode] ?? normalizeAreaLabel(document.processArea ?? areaCode));
    const currentAreas = profileAreaCountByCode[profileCode] ?? {};
    currentAreas[areaLabel] = (currentAreas[areaLabel] ?? 0) + 1;
    profileAreaCountByCode[profileCode] = currentAreas;
  }

  return { profileCountByCode, profileAreaCountByCode };
}

export function buildDocumentProfileCountMap(documents: SearchDocumentItem[]): Record<string, number> {
  return aggregateDocuments(documents, []).profileCountByCode;
}

export function buildProfileAccordions(
  profiles: DocumentProfileItem[],
  documents: SearchDocumentItem[],
  processAreas: ProcessAreaItem[],
): ProfileAccordionSummary[] {
  const aggregate = aggregateDocuments(documents, processAreas);
  return profiles.map((profile) => {
    const areas = aggregate.profileAreaCountByCode[profile.code] ?? {};
    return {
      code: profile.code,
      label: profile.alias || profile.name,
      count: aggregate.profileCountByCode[profile.code] ?? 0,
      areas: Object.entries(areas)
        .map(([label, count]) => ({ label, count }))
        .sort((left, right) => right.count - left.count)
        .slice(0, 5),
    };
  });
}
