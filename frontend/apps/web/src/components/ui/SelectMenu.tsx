import { useEffect, useMemo, useState, type FocusEvent } from "react";

export type SelectMenuOption = {
  label: string;
  value: string;
};

type SelectMenuProps = {
  id: string;
  value: string;
  values?: string[];
  mode?: "single" | "multi";
  options: SelectMenuOption[];
  onSelect: (value: string) => void;
  label?: string;
  placeholder?: string;
  disabled?: boolean;
  chevronStrokeWidth?: number;
  searchThreshold?: number;
  closeOnMultiSelect?: boolean;
};

export function SelectMenu(props: SelectMenuProps) {
  const {
    id,
    value,
    values = [],
    mode = "single",
    options,
    onSelect,
    label,
    placeholder,
    disabled = false,
    chevronStrokeWidth = 2,
    searchThreshold = 10,
    closeOnMultiSelect = false,
  } = props;
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const selectedValues = mode === "multi" ? values : [value];
  const showSearch = options.length > Math.max(0, Number(searchThreshold) || 0);
  const visibleOptions = useMemo(() => {
    if (!showSearch) return options;
    const token = searchQuery.trim().toLowerCase();
    if (!token) return options;
    return options.filter((option) => {
      const optionLabel = String(option.label || "").toLowerCase();
      const optionValue = String(option.value || "").toLowerCase();
      return optionLabel.includes(token) || optionValue.includes(token);
    });
  }, [options, searchQuery, showSearch]);
  const currentLabel = (() => {
    if (mode !== "multi") {
      return options.find((option) => option.value === value)?.label ?? placeholder ?? "";
    }
    if (!selectedValues.length) return placeholder ?? options.find((option) => option.value === "all")?.label ?? "";
    if (selectedValues.length === 1) return options.find((option) => option.value === selectedValues[0])?.label ?? selectedValues[0];
    return `${selectedValues.length} selecionadas`;
  })();

  function handleBlur(event: FocusEvent<HTMLDivElement>) {
    const nextTarget = event.relatedTarget as Node | null;
    if (!nextTarget || !event.currentTarget.contains(nextTarget)) {
      setOpen(false);
    }
  }

  useEffect(() => {
    if (!open && searchQuery) setSearchQuery("");
  }, [open, searchQuery]);

  return (
    <div className="spotlight-select" onBlur={handleBlur} data-open={open ? "true" : "false"}>
      {label ? (
        <label htmlFor={id} className="spotlight-select-label">
          {label}
        </label>
      ) : null}
      <button
        id={id}
        type="button"
        className="spotlight-select-trigger"
        onClick={() => {
          if (!disabled) setOpen((prev) => !prev);
        }}
        onKeyDown={(event) => {
          if (event.key === "Escape") setOpen(false);
        }}
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={disabled}
      >
        <span className="spotlight-select-value">{currentLabel}</span>
        <span className="spotlight-select-chevron" aria-hidden>
          <svg viewBox="0 0 20 20" fill="none">
            <path
              d="M5 7.5L10 12.5L15 7.5"
              stroke="currentColor"
              strokeWidth={chevronStrokeWidth}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </span>
      </button>
      {open ? (
        <div className="spotlight-select-menu" role="listbox" aria-labelledby={id}>
          {showSearch ? (
            <div className="spotlight-select-search">
              <input
                type="search"
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                onMouseDown={(event) => event.stopPropagation()}
                placeholder="Pesquisar..."
                className="spotlight-select-search-input"
              />
            </div>
          ) : null}
          {visibleOptions.map((option) => {
            const isActive =
              mode === "multi"
                ? option.value === "all"
                  ? selectedValues.length === 0
                  : selectedValues.includes(option.value)
                : option.value === value;
            return (
              <button
                key={`${id}-${option.value || "all"}`}
                type="button"
                className={`spotlight-select-option${isActive ? " is-active" : ""}`}
                role="option"
                aria-selected={isActive}
                onMouseDown={(event) => event.preventDefault()}
                onClick={(event) => {
                  event.stopPropagation();
                  onSelect(option.value);
                  if (mode !== "multi" || option.value === "all" || closeOnMultiSelect) {
                    setOpen(false);
                  }
                }}
              >
                {option.label}
              </button>
            );
          })}
          {visibleOptions.length === 0 ? <div className="spotlight-select-empty">Nenhum resultado</div> : null}
        </div>
      ) : null}
    </div>
  );
}
