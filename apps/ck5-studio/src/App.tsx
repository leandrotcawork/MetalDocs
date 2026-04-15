import { useState } from "react";
import { EditorCanvas } from "./components/EditorCanvas";

const INITIAL_HTML = "<h1>Untitled Editorial Concept</h1><p>Start writing here.</p>";

export default function App() {
  const [html, setHtml] = useState(INITIAL_HTML);
  return <EditorCanvas initialData={html} onChange={setHtml} />;
}
