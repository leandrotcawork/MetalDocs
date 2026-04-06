/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_CKEDITOR_LICENSE_KEY: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
