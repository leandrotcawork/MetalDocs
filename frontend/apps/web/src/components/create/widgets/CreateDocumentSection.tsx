import type { ReactNode } from "react";

type CreateDocumentSectionProps = {
  sectionId?: string;
  title: string;
  subtitle: string;
  icon: ReactNode;
  children: ReactNode;
};

export function CreateDocumentSection(props: CreateDocumentSectionProps) {
  return (
    <section id={props.sectionId} className="create-doc-section">
      <header className="create-doc-section-head">
        <span className="create-doc-section-icon" aria-hidden="true">{props.icon}</span>
        <div>
          <h3>{props.title}</h3>
          <small>{props.subtitle}</small>
        </div>
      </header>
      <div className="create-doc-section-body">{props.children}</div>
    </section>
  );
}
