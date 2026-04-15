import type { ReactNode } from "react";
import type { LibraryItemKey, SelectionSummary } from "../types";
import { TopBar } from "./TopBar";
import { LeftLibrary } from "./LeftLibrary";
import { RightPanel } from "./RightPanel";

type AppShellProps = {
  children: ReactNode;
  selection: SelectionSummary | null;
  onInsert: (key: LibraryItemKey) => void;
  onImagePick: () => void;
};

export function AppShell({ children, selection, onInsert, onImagePick }: AppShellProps) {
  return (
    <div className="studio-shell">
      <TopBar />
      <div className="studio-body">
        <LeftLibrary onInsert={onInsert} onImagePick={onImagePick} />
        <main className="studio-center" data-testid="editor-canvas">
          {children}
        </main>
        <RightPanel selection={selection} />
      </div>
    </div>
  );
}
