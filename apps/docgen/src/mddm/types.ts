export type MDDMBlock = {
  id: string;
  template_block_id?: string;
  type: string;
  props: Record<string, unknown>;
  children?: MDDMBlock[] | InlineRun[];
};

export type InlineRun = {
  text: string;
  marks?: { type: string }[];
  link?: { href: string; title?: string };
  document_ref?: { target_document_id: string; target_revision_label?: string };
};

export type MDDMEnvelope = {
  mddm_version: number;
  blocks: MDDMBlock[];
  template_ref: any;
};

export type MDDMExportRequest = {
  envelope: MDDMEnvelope;
  metadata: {
    document_code: string;
    title: string;
    revision_label: string;
    mode: "production" | "debug";
  };
};
