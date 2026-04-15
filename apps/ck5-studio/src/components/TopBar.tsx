export function TopBar() {
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
