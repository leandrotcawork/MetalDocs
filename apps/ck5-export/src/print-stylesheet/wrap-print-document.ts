import { PRINT_STYLESHEET } from "./print-css";

/**
 * Wraps raw HTML body content in a full print-ready HTML document.
 * Caller is responsible for ensuring bodyHtml is sanitized — it is
 * interpolated directly into the document string.
 */
export function wrapInPrintDocument(bodyHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>MDDM Document</title>
<style>${PRINT_STYLESHEET}</style>
<script src="/assets/paged.polyfill.js" defer></script>
</head>
<body>
${bodyHtml}
</body>
</html>`;
}
