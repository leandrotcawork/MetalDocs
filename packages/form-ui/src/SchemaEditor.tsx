import Editor from '@monaco-editor/react';
import Ajv from 'ajv';

const ajv = new Ajv({ allErrors: true, strict: false });

export interface SchemaEditorProps {
  value: string;
  onChange: (v: string) => void;
  height?: number | string;
}

export function SchemaEditor(props: SchemaEditorProps) {
  return (
    <Editor
      height={props.height ?? '100%'}
      defaultLanguage="json"
      value={props.value}
      onChange={(v) => props.onChange(v ?? '')}
      options={{ minimap: { enabled: false }, fontSize: 13, automaticLayout: true }}
    />
  );
}

export function validateJsonSchema(raw: string): { valid: boolean; errors: string[] } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (e) {
    return { valid: false, errors: [`JSON parse: ${(e as Error).message}`] };
  }
  try {
    ajv.compile(parsed as object);
    return { valid: true, errors: [] };
  } catch (e) {
    return { valid: false, errors: [String((e as Error).message)] };
  }
}
