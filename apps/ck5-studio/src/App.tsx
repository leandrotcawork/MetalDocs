import { useRef, useState } from "react";
import { AppShell } from "./components/AppShell";
import { EditorCanvas } from "./components/EditorCanvas";
import type { LibraryItemKey, SelectionSummary } from "./types";
import { DEFAULT_EDITORIAL_HTML, snippetFor } from "./lib/contentSnippets";
import type { EditingTemplateMode } from "./lib/editorConfig";

type InsertCommand = {
  id: number;
  html: string;
};

export default function App() {
  const [html, setHtml] = useState(DEFAULT_EDITORIAL_HTML);
  const [selection] = useState<SelectionSummary | null>(null);
  const [mode, setMode] = useState<EditingTemplateMode>("author");
  const [insertCommand, setInsertCommand] = useState<InsertCommand | null>(null);
  const commandSeqRef = useRef(0);

  function queueInsert(htmlToInsert: string) {
    commandSeqRef.current += 1;
    const next = commandSeqRef.current;
    setInsertCommand({ id: next, html: htmlToInsert });
  }

  function handleNewDocument() {
    setInsertCommand(null);
    setHtml(DEFAULT_EDITORIAL_HTML);
  }

  function handleInsert(key: LibraryItemKey) {
    if (mode === "fill" || key === "image") {
      return;
    }
    queueInsert(snippetFor(key));
  }

  function handleImagePick() {
    if (mode === "fill") {
      return;
    }
    queueInsert("<p>[ Image placeholder ]</p><p><em>Add media asset...</em></p>");
  }

  return (
    <AppShell
      selection={selection}
      mode={mode}
      onModeChange={setMode}
      onNewDocument={handleNewDocument}
      onInsert={handleInsert}
      onImagePick={handleImagePick}
    >
      <EditorCanvas
        initialData={html}
        mode={mode}
        insertCommand={insertCommand}
        onInsertApplied={() => setInsertCommand(null)}
        onChange={setHtml}
        onDebouncedChange={setHtml}
      />
    </AppShell>
  );
}
