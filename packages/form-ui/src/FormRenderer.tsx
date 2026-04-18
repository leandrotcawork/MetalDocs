import Form from '@rjsf/core';
import validator from '@rjsf/validator-ajv8';
import type { RJSFSchema } from '@rjsf/utils';

export interface FormRendererProps {
  schema: RJSFSchema;
  formData: unknown;
  onChange: (data: unknown) => void;
  onSubmit?: (data: unknown) => void;
  disabled?: boolean;
}

export function FormRenderer(props: FormRendererProps) {
  return (
    <Form
      schema={props.schema}
      formData={props.formData}
      validator={validator}
      onChange={(e) => props.onChange(e.formData)}
      onSubmit={(e) => props.onSubmit?.(e.formData)}
      disabled={props.disabled}
      showErrorList={false}
      liveValidate
    />
  );
}
