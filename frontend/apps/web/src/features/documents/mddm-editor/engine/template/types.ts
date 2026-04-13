export type TemplateStatus = "draft" | "published" | "deprecated";

export type TemplateMeta = {
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};

export type TemplateTheme = {
  accent: string;
  accentLight: string;
  accentDark: string;
  accentBorder: string;
};

export type TemplateBlock = {
  type: string;
  props: Record<string, unknown>;
  style?: Record<string, unknown>;
  capabilities?: Record<string, unknown>;
  columns?: Array<{ key: string; label: string; width: string; locked?: boolean }>;
  content?: unknown;
  children?: TemplateBlock[];
};

export type TemplateDefinition = {
  templateKey: string;
  version: number;
  profileCode: string;
  status: TemplateStatus;
  meta: TemplateMeta;
  theme: TemplateTheme;
  blocks: TemplateBlock[];
};

export type TemplateRef = {
  templateKey: string;
  templateVersion: number;
  instantiatedAt: string;
};