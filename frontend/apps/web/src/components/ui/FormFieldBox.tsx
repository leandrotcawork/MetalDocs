import { FilterDropdown, type SelectMenuOption } from "./FilterDropdown";
import styles from "./FormFieldBox.module.css";

type TextFieldBoxProps = {
  id: string;
  label?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "text" | "email" | "password" | "search";
  readOnly?: boolean;
};

type DropdownFieldBoxProps = {
  id: string;
  label?: string;
  value: string;
  options: SelectMenuOption[];
  onSelect: (value: string) => void;
  placeholder?: string;
};

export function TextFieldBox(props: TextFieldBoxProps) {
  return (
    <div className={styles.field}>
      {props.label ? <span className={styles.label}>{props.label}</span> : null}
      <input
        id={props.id}
        className={styles.input}
        type={props.type ?? "text"}
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        placeholder={props.placeholder}
        readOnly={props.readOnly}
      />
    </div>
  );
}

export function DropdownFieldBox(props: DropdownFieldBoxProps) {
  return (
    <div className={styles.field}>
      {props.label ? <span className={styles.label}>{props.label}</span> : null}
      <FilterDropdown
        id={props.id}
        value={props.value}
        options={props.options}
        onSelect={props.onSelect}
        placeholder={props.placeholder}
      />
    </div>
  );
}
