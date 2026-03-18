import { useEffect, useMemo, useRef, useState } from "react";

type TopbarDropdownOption = {
  label: string;
  value: string;
};

type TopbarDropdownProps = {
  id: string;
  value: string;
  options: TopbarDropdownOption[];
  onChange: (value: string) => void;
};

export function TopbarDropdown(props: TopbarDropdownProps) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement | null>(null);

  const currentLabel = useMemo(() => {
    return props.options.find((item) => item.value === props.value)?.label ?? props.options[0]?.label ?? "";
  }, [props.options, props.value]);

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!wrapRef.current) {
        return;
      }
      if (!wrapRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, []);

  return (
    <div ref={wrapRef} className="catalog-dropdown" data-open={open ? "true" : "false"}>
      <button
        id={props.id}
        type="button"
        className="catalog-dropdown-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="catalog-dropdown-value">{currentLabel}</span>
        <span className="catalog-dropdown-chevron" aria-hidden="true">
          <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.4">
            <path d="M3.5 5 6.5 8 9.5 5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </span>
      </button>

      {open && (
        <div className="catalog-dropdown-menu" role="listbox" aria-labelledby={props.id}>
          {props.options.map((item) => (
            <button
              key={`${props.id}-${item.value}`}
              type="button"
              className={`catalog-dropdown-option ${item.value === props.value ? "is-active" : ""}`}
              role="option"
              aria-selected={item.value === props.value}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => {
                props.onChange(item.value);
                setOpen(false);
              }}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
