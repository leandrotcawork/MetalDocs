import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    // Force @eigenpal/docx-js-editor (and its deep deps) through Vite's
    // transform pipeline so that CSS side-effect imports don't crash Node.
    server: {
      deps: {
        inline: [/@eigenpal\/docx-js-editor/],
      },
    },
  },
  plugins: [
    {
      name: 'css-stub',
      enforce: 'pre',
      transform(_code, id) {
        if (id.endsWith('.css')) return { code: '', map: null };
        return undefined;
      },
      load(id) {
        if (id.endsWith('.css')) return '';
        return undefined;
      },
    },
  ],
});
