import type { ReactNode } from "react";

type CreateFieldProps = {
  label: string;
  required?: boolean;
  hint?: string;
  children: ReactNode;
};

export function CreateField(props: CreateFieldProps) {
  return (
    <div className="field">
      <label className="field-label">
        <span>{props.label}</span>
        {props.required ? <span className="field-required">*</span> : null}
      </label>
      {props.children}
      <small className="field-hint">{props.hint ?? ""}</small>
    </div>
  );
}
