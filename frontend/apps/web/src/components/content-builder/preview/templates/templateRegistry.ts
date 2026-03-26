import type { PreviewTemplateProps } from "./PreviewTemplateGeneric";
import { PreviewTemplatePO } from "./PreviewTemplatePO";
import { PreviewTemplateIT } from "./PreviewTemplateIT";
import { PreviewTemplateRG } from "./PreviewTemplateRG";
import { PreviewTemplateFM } from "./PreviewTemplateFM";

type TemplateComponent = (props: PreviewTemplateProps) => React.JSX.Element;

const registry: Record<string, TemplateComponent> = {
  po: PreviewTemplatePO,
  it: PreviewTemplateIT,
  rg: PreviewTemplateRG,
  fm: PreviewTemplateFM,
};

export function getPreviewTemplate(profileCode: string): TemplateComponent | null {
  return registry[profileCode.toLowerCase()] ?? null;
}
