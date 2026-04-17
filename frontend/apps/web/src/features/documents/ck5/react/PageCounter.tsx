import { useEffect, useState } from 'react';
import type { ClassicEditor } from 'ckeditor5';

type ComputedBreak = { afterBid: string; pageNumber: number };

export function PageCounter({ editor }: { editor: ClassicEditor | null }) {
  const [pages, setPages] = useState(1);

  useEffect(() => {
    if (!editor) return;
    const plugin = editor.plugins.get('MddmPagination') as any;
    if (!plugin?._measurer) return;
    const off = plugin._measurer.onBreaks((b: ComputedBreak[]) => {
      // pages = last break's pageNumber; or 1 if no breaks
      setPages(b.length ? b[b.length - 1].pageNumber : 1);
    });
    return () => off();
  }, [editor]);

  return <span className="mddm-page-counter">Page {pages} of {pages}</span>;
}
