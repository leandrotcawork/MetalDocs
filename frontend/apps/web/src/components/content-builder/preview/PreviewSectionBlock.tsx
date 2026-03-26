import type { ReactNode } from "react";

type PreviewSectionBlockProps = {
  index: number;
  title: string;
  description?: string;
  sectionKey: string;
  children: ReactNode;
};

export function PreviewSectionBlock({ index, title, description, sectionKey, children }: PreviewSectionBlockProps) {
  return (
    <section className="preview-section" data-preview-section={sectionKey}>
      <div className="preview-section-header">
        <h3 className="preview-section-title">
          <span className="preview-section-number">{index + 1}.</span>
          {title}
        </h3>
        {description && <p className="preview-section-description">{description}</p>}
      </div>
      <div className="preview-section-body">{children}</div>
    </section>
  );
}
