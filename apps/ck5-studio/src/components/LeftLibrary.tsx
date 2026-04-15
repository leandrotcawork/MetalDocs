import type { LibraryItemKey } from "../types";

type LeftLibraryProps = {
  onInsert: (key: LibraryItemKey) => void;
  onImagePick: () => void;
};

export function LeftLibrary({ onInsert, onImagePick }: LeftLibraryProps) {
  return (
    <aside className="studio-left-panel" data-testid="left-library">
      <div className="panel-kicker">Library</div>
      <div className="panel-subtle">Intellectual Atelier</div>
      <button type="button" className="panel-cta" onClick={() => onInsert("text")}>
        + New Document
      </button>
      <div className="library-group-label">Basic Blocks</div>
      <button type="button" className="library-tile" onClick={() => onInsert("text")}>
        Text
      </button>
      <button type="button" className="library-tile" onClick={onImagePick}>
        Media
      </button>
      <button type="button" className="library-tile" onClick={() => onInsert("table")}>
        Table
      </button>
      <button type="button" className="library-tile" onClick={() => onInsert("section")}>
        Section
      </button>
      <div className="library-group-label">Structural</div>
      <button type="button" className="library-row" onClick={() => onInsert("heading")}>
        Heading
      </button>
    </aside>
  );
}
