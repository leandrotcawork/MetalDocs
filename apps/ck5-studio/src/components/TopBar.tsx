type TopBarProps = {
  mode: "author" | "fill";
  onModeChange: (mode: "author" | "fill") => void;
};

export function TopBar({ mode, onModeChange }: TopBarProps) {
  return (
    <header className="studio-topbar" data-testid="top-bar">
      <div className="studio-brand">The Editorial Architect</div>
      <nav className="studio-nav">
        <a href="#" className="is-muted">
          Documents
        </a>
        <a href="#" className="is-active">
          Templates
        </a>
        <a href="#" className="is-muted">
          Publish
        </a>
      </nav>
      <div className="studio-actions">
        <button
          type="button"
          className={`topbar-mode-btn ${mode === "author" ? "is-active" : ""}`}
          onClick={() => onModeChange("author")}
        >
          Author Mode
        </button>
        <button
          type="button"
          className={`topbar-mode-btn ${mode === "fill" ? "is-active" : ""}`}
          onClick={() => onModeChange("fill")}
        >
          Fill Mode
        </button>
        <span className="topbar-mode-hint" data-testid="mode-hint">
          {mode === "fill" ? "Fill mode: edit highlighted fields only" : "Author mode: full template editing"}
        </span>
        <button type="button" className="topbar-ghost-btn">
          Preview
        </button>
        <button type="button" className="primary-btn">
          Share
        </button>
      </div>
    </header>
  );
}
