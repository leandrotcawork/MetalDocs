import { useState } from "react";
import { AppShell } from "./components/AppShell";
import { EditorCanvas } from "./components/EditorCanvas";
import type { LibraryItemKey, SelectionSummary } from "./types";

const INITIAL_HTML = "<h1>Untitled Editorial Concept</h1><p>Start writing here.</p>";

export default function App() {
  const [html, setHtml] = useState(INITIAL_HTML);
  const [selection] = useState<SelectionSummary | null>(null);

  function handleInsert(_key: LibraryItemKey) {}

  return (
    <AppShell selection={selection} onInsert={handleInsert} onImagePick={() => {}}>
      <EditorCanvas initialData={html} onDebouncedChange={setHtml} />
    </AppShell>
  );
}
