import { useState } from 'react';
import { ExportMenu } from './ExportMenu';

export function ExportMenuButton({ documentID, canExport }: { documentID: string; canExport: boolean }) {
  const [open, setOpen] = useState(false);

  return (
    <details open={open} onToggle={(event) => setOpen((event.currentTarget as HTMLDetailsElement).open)}>
      <summary>Export</summary>
      <ExportMenu documentID={documentID} canExport={canExport} />
    </details>
  );
}
