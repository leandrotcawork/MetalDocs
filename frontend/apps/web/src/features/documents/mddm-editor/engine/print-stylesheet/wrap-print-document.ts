import { PRINT_STYLESHEET } from "./print-css";

export function wrapInPrintDocument(bodyHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>MDDM Document</title>
<style>${PRINT_STYLESHEET}</style>
</head>
<body>
${bodyHtml}
</body>
</html>`;
}
