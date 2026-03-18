import type { DocumentProfileItem, SearchDocumentItem } from "../../../lib.types";

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

function aggregateDocuments(documents: SearchDocumentItem[]): DocumentAggregate {
  const profileCountByCode: Record<string, number> = {};
  const profileAreaCountByCode: Record<string, Record<string, number>> = {};

  for (const document of documents) {
    const profileCode = document.documentProfile;
    if (!profileCode) {
      continue;
    }

    profileCountByCode[profileCode] = (profileCountByCode[profileCode] ?? 0) + 1;

    const areaLabel = document.processArea || "Sem area";
    const currentAreas = profileAreaCountByCode[profileCode] ?? {};
    currentAreas[areaLabel] = (currentAreas[areaLabel] ?? 0) + 1;
    profileAreaCountByCode[profileCode] = currentAreas;
  }

  return { profileCountByCode, profileAreaCountByCode };
}

export function buildDocumentProfileCountMap(documents: SearchDocumentItem[]): Record<string, number> {
  return aggregateDocuments(documents).profileCountByCode;
}

export function buildProfileAccordions(
  profiles: DocumentProfileItem[],
  documents: SearchDocumentItem[],
): ProfileAccordionSummary[] {
  const aggregate = aggregateDocuments(documents);
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
