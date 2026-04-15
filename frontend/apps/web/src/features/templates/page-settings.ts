export type TemplatePageSettings = {
  marginTopMm: number;
  marginRightMm: number;
  marginBottomMm: number;
  marginLeftMm: number;
};

export const defaultTemplatePageSettings: TemplatePageSettings = {
  marginTopMm: 25,
  marginRightMm: 20,
  marginBottomMm: 25,
  marginLeftMm: 25,
};

const PAGE_MARGIN_MIN_MM = 5;
const PAGE_MARGIN_MAX_MM = 50;

type UnknownRecord = Record<string, unknown>;

function isRecord(value: unknown): value is UnknownRecord {
  return typeof value === "object" && value !== null;
}

function readMarginValue(page: UnknownRecord, key: keyof TemplatePageSettings, fallback: number): number {
  const value = page[key];
  return clampPageMarginMm(value as number, fallback);
}

export function clampPageMarginMm(value: number, fallback: number = defaultTemplatePageSettings.marginTopMm): number {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.min(PAGE_MARGIN_MAX_MM, Math.max(PAGE_MARGIN_MIN_MM, value));
}

export function readTemplatePageSettings(meta: unknown): TemplatePageSettings {
  if (!isRecord(meta) || !isRecord(meta.page)) {
    return { ...defaultTemplatePageSettings };
  }

  const page = meta.page;
  return {
    marginTopMm: readMarginValue(page, "marginTopMm", defaultTemplatePageSettings.marginTopMm),
    marginRightMm: readMarginValue(page, "marginRightMm", defaultTemplatePageSettings.marginRightMm),
    marginBottomMm: readMarginValue(page, "marginBottomMm", defaultTemplatePageSettings.marginBottomMm),
    marginLeftMm: readMarginValue(page, "marginLeftMm", defaultTemplatePageSettings.marginLeftMm),
  };
}

export function writeTemplatePageSettings(meta: unknown, page: TemplatePageSettings): Record<string, unknown> {
  const base = isRecord(meta) ? meta : {};
  const basePage = isRecord(base.page) ? base.page : {};
  return {
    ...base,
    page: {
      ...basePage,
      marginTopMm: clampPageMarginMm(page.marginTopMm, defaultTemplatePageSettings.marginTopMm),
      marginRightMm: clampPageMarginMm(page.marginRightMm, defaultTemplatePageSettings.marginRightMm),
      marginBottomMm: clampPageMarginMm(page.marginBottomMm, defaultTemplatePageSettings.marginBottomMm),
      marginLeftMm: clampPageMarginMm(page.marginLeftMm, defaultTemplatePageSettings.marginLeftMm),
    },
  };
}
