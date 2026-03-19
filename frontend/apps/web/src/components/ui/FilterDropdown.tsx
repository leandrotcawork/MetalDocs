import type { SelectMenuOption } from "./SelectMenu";
import { SelectMenu } from "./SelectMenu";

type FilterDropdownProps = {
  id: string;
  options: SelectMenuOption[];
  onSelect: (value: string) => void;
  label?: string;
  value?: string;
  values?: string[];
  selectionMode?: "one" | "duo";
  disabled?: boolean;
  searchThreshold?: number;
  closeOnSelectInDuo?: boolean;
  chevronStrokeWidth?: number;
};

export type { SelectMenuOption };

export function FilterDropdown({
  id,
  options,
  onSelect,
  label,
  value = "",
  values = [],
  selectionMode = "one",
  disabled = false,
  searchThreshold = 10,
  closeOnSelectInDuo = false,
  chevronStrokeWidth = 2,
}: FilterDropdownProps) {
  return (
    <SelectMenu
      id={id}
      label={label}
      options={options}
      onSelect={onSelect}
      value={value}
      values={values}
      mode={selectionMode === "duo" ? "multi" : "single"}
      disabled={disabled}
      searchThreshold={searchThreshold}
      closeOnMultiSelect={closeOnSelectInDuo}
      chevronStrokeWidth={chevronStrokeWidth}
    />
  );
}
