import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const stubCss = {
  name: 'stub-css',
  load(id: string) {
    if (id.endsWith('.css')) return { code: 'export default {}', map: null };
  },
};

export default defineConfig({
  plugins: [stubCss, react()],
  test: {
    environment: 'jsdom',
    server: {
      deps: {
        inline: [/prosemirror-view/, /@eigenpal/],
      },
    },
  },
});
