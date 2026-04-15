import type { ReactNode } from "react";
import type { LibraryItemKey, SelectionSummary } from "../types";
import { TopBar } from "./TopBar";
import { LeftLibrary } from "./LeftLibrary";
import { RightPanel } from "./RightPanel";

type AppShellProps = {
  children: ReactNode;
  selection: SelectionSummary | null;
  mode: "author" | "fill";
  onModeChange: (mode: "author" | "fill") => void;
  onNewDocument: () => void;
  onInsert: (key: LibraryItemKey) => void;
  onImagePick: () => void;
};

export function AppShell({
  children,
  selection,
  mode,
  onModeChange,
  onNewDocument,
  onInsert,
  onImagePick,
}: AppShellProps) {
  return (
    <div className="studio-shell">
      <TopBar mode={mode} onModeChange={onModeChange} />
      <div className="studio-body">
        <LeftLibrary
          onNewDocument={onNewDocument}
          onInsert={onInsert}
          onImagePick={onImagePick}
          actionsDisabled={mode === "fill"}
        />
        <main className="studio-center" data-testid="editor-canvas">
          {children}
        </main>
        <RightPanel selection={selection} />
      </div>
    </div>
  );
}
