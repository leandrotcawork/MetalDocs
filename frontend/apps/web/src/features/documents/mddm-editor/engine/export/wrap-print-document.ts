import { PRINT_STYLESHEET } from "../print-stylesheet";

export function wrapInPrintDocument(bodyHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<title>MDDM Document</title>
<style>${PRINT_STYLESHEET}</style>
</head>
<body>
${bodyHtml}
</body>
</html>`;
}
