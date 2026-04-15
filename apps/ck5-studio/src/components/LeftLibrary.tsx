import type { LibraryItemKey } from "../types";

type LeftLibraryProps = {
  onNewDocument: () => void;
  onInsert: (key: LibraryItemKey) => void;
  onImagePick: () => void;
  actionsDisabled?: boolean;
};

export function LeftLibrary({ onNewDocument, onInsert, onImagePick, actionsDisabled = false }: LeftLibraryProps) {
  return (
    <aside className="studio-left-panel" data-testid="left-library">
      <div className="panel-kicker">Library</div>
      <div className="panel-subtle">Intellectual Atelier</div>
      <button type="button" className="panel-cta" onClick={onNewDocument} disabled={actionsDisabled}>
        + New Document
      </button>
      <div className="library-group-label">Basic Blocks</div>
      <div className="library-grid">
        <button type="button" className="library-tile" onClick={() => onInsert("text")} disabled={actionsDisabled}>
          Text
        </button>
        <button type="button" className="library-tile" onClick={onImagePick} disabled={actionsDisabled}>
          Media
        </button>
        <button type="button" className="library-tile" onClick={() => onInsert("table")} disabled={actionsDisabled}>
          Table
        </button>
        <button type="button" className="library-tile" onClick={() => onInsert("section")} disabled={actionsDisabled}>
          Section
        </button>
        <button type="button" className="library-tile" onClick={() => onInsert("note")} disabled={actionsDisabled}>
          Block Note
        </button>
        <button type="button" className="library-tile" onClick={() => onInsert("mixed")} disabled={actionsDisabled}>
          Mixed Section
        </button>
      </div>
      <div className="library-group-label">Structural</div>
      <div className="library-list">
        <button type="button" className="library-row" onClick={() => onInsert("heading")} disabled={actionsDisabled}>
          Heading
        </button>
      </div>
    </aside>
  );
}
