import { SubBlockRegistry } from "./registry";
import { DocHeaderStandard } from "./doc_header_standard";
import { RevisionBox } from "./revision_box";
import { ApprovalSignaturesBlock } from "./approval_signatures_block";
import { FooterPageNumbers } from "./footer_page_numbers";
import { FooterControlledCopyNotice } from "./footer_controlled_copy_notice";

export function registerV1Builtins(r: SubBlockRegistry): void {
  r.register(DocHeaderStandard);
  r.register(RevisionBox);
  r.register(ApprovalSignaturesBlock);
  r.register(FooterPageNumbers);
  r.register(FooterControlledCopyNotice);
}
