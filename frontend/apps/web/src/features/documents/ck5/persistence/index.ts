import * as local from './localStorageStub';
import * as api from './apiPersistence';

const mode = import.meta.env.VITE_CK5_PERSISTENCE ?? 'local';
const impl = mode === 'api' ? api : (local as unknown as typeof api);

export const { saveTemplate, loadTemplate, saveDocument, loadDocument } = impl;
export type { TemplateRecord } from './localStorageStub';
