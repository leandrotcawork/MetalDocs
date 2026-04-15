import type { SelectionSummary } from "../types";

type RightPanelProps = {
  selection: SelectionSummary | null;
};

export function RightPanel({ selection }: RightPanelProps) {
  return (
    <aside className="studio-right-panel" data-testid="right-panel">
      <div className="panel-title-row">
        <h2>Properties</h2>
        <span className="panel-chip">{selection?.label ?? "DOCUMENT"}</span>
      </div>
      <div className="panel-meta">Element: {selection?.elementTag ?? "body"}</div>
      <div className="panel-section-title">Dimensions</div>
      <div className="panel-card">
        <span>Width</span>
        <strong>100%</strong>
      </div>
      <div className="panel-section-title">Background Surface</div>
      <div className="color-row">
        <span className="color-dot is-paper" />
        <span className="color-dot is-maroon" />
        <span className="color-dot is-charcoal" />
        <span className="color-dot is-blush" />
      </div>
      <div className="panel-section-title">Spacing</div>
      <div className="panel-card is-spacing-card">32 / 48 / 32 / 48</div>
      <button type="button" className="primary-btn panel-save">
        Save Template
      </button>
      <button type="button" className="ghost-link">
        Discard Changes
      </button>
    </aside>
  );
}
