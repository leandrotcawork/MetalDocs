import { SubBlockRenderer, SubBlockContext } from "./registry";

type Approver = { user_id: unknown; display_name: unknown; signed_at: unknown };

function str(v: unknown): string {
  if (v === null || v === undefined) return "";
  return String(v);
}

function cell(text: string): string {
  return `<w:tc><w:p><w:r><w:t xml:space="preserve">${text}</w:t></w:r></w:p></w:tc>`;
}

function row(a: string, b: string): string {
  return `<w:tr>${cell(a)}${cell(b)}</w:tr>`;
}

function isApprover(v: unknown): v is Approver {
  return typeof v === "object" && v !== null;
}

export const ApprovalSignaturesBlock: SubBlockRenderer = {
  key: "approval_signatures_block",
  async render(ctx: SubBlockContext): Promise<string> {
    const raw = ctx.values.approvers;
    const approvers: Approver[] = Array.isArray(raw) ? raw.filter(isApprover) : [];

    const header = row("Name", "Signed At");
    const body = approvers
      .map((a) => row(str(a.display_name), str(a.signed_at)))
      .join("");

    return `<w:tbl>${header}${body}</w:tbl>`;
  },
};
