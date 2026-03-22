import type { ReactNode } from "react";
import type { StepStatus } from "../documentCreateTypes";

export type ProgressSidebarItem = {
  key: string;
  label: string;
  description?: string;
  status: StepStatus;
  isCurrent?: boolean;
  onSelect: () => void;
};

type ProgressSidebarProps = {
  title?: string;
  items: ProgressSidebarItem[];
};

function glyphFor(status: StepStatus, index: number): ReactNode {
  if (status === "done") {
    return (
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M3 8.5 6.3 11.5 13 4.8" />
      </svg>
    );
  }
  if (status === "error") {
    return (
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M4.2 4.2 11.8 11.8M11.8 4.2 4.2 11.8" />
      </svg>
    );
  }
  return index + 1;
}

export function ProgressSidebar(props: ProgressSidebarProps) {
  return (
    <aside className="create-doc-steps" aria-label={props.title ?? "Progresso"}>
      <p className="create-doc-steps-title">{props.title ?? "Etapas"}</p>
      <div className="create-doc-steps-list">
        {props.items.map((item, index) => (
          <div key={item.key} className="create-doc-step-row">
            <button
              type="button"
              className="create-doc-step-item"
              data-status={item.status}
              aria-current={item.isCurrent ? "step" : undefined}
              onClick={item.onSelect}
            >
              <span className="create-doc-step-num">
                <span className="create-doc-step-glyph">{glyphFor(item.status, index)}</span>
              </span>
              <div>
                <strong>{item.label}</strong>
                {item.description && <small>{item.description}</small>}
              </div>
            </button>
            {index < props.items.length - 1 && (
              <div className="create-doc-step-connector" data-status={item.status} aria-hidden="true" />
            )}
          </div>
        ))}
      </div>
    </aside>
  );
}
