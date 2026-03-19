import { useEffect, useMemo, useState, type FocusEvent } from "react";

type DateTimeFieldProps = {
  id: string;
  value: string;
  onChange: (nextValue: string) => void;
  placeholder?: string;
};

type DateParts = {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
};

const monthNames = [
  "janeiro",
  "fevereiro",
  "marco",
  "abril",
  "maio",
  "junho",
  "julho",
  "agosto",
  "setembro",
  "outubro",
  "novembro",
  "dezembro",
];

function pad2(value: number) {
  return value.toString().padStart(2, "0");
}

function parseValue(value: string): DateParts | null {
  if (!value) return null;
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/);
  if (!match) return null;
  const [, year, month, day, hour, minute] = match;
  return {
    year: Number(year),
    month: Number(month),
    day: Number(day),
    hour: Number(hour),
    minute: Number(minute),
  };
}

function toValueString(parts: DateParts) {
  return `${parts.year}-${pad2(parts.month)}-${pad2(parts.day)}T${pad2(parts.hour)}:${pad2(parts.minute)}`;
}

function formatDisplay(parts: DateParts | null) {
  if (!parts) return "";
  return `${pad2(parts.day)}/${pad2(parts.month)}/${parts.year} ${pad2(parts.hour)}:${pad2(parts.minute)}`;
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

export function DateTimeField(props: DateTimeFieldProps) {
  const [open, setOpen] = useState(false);
  const [viewYear, setViewYear] = useState<number>(() => new Date().getFullYear());
  const [viewMonth, setViewMonth] = useState<number>(() => new Date().getMonth() + 1);
  const parsed = useMemo(() => parseValue(props.value), [props.value]);
  const selected = parsed ?? null;

  useEffect(() => {
    if (!open) return;
    const basis = selected ?? {
      year: new Date().getFullYear(),
      month: new Date().getMonth() + 1,
      day: new Date().getDate(),
      hour: 9,
      minute: 0,
    };
    setViewYear(basis.year);
    setViewMonth(basis.month);
  }, [open, selected]);

  function ensureParts(): DateParts {
    const now = new Date();
    return selected ?? {
      year: now.getFullYear(),
      month: now.getMonth() + 1,
      day: now.getDate(),
      hour: 9,
      minute: 0,
    };
  }

  function updateParts(patch: Partial<DateParts>) {
    const next = { ...ensureParts(), ...patch };
    props.onChange(toValueString(next));
  }

  function handleBlur(event: FocusEvent<HTMLDivElement>) {
    const nextTarget = event.relatedTarget as Node | null;
    if (!nextTarget || !event.currentTarget.contains(nextTarget)) {
      setOpen(false);
    }
  }

  const daysGrid = useMemo(() => {
    const firstDay = new Date(viewYear, viewMonth - 1, 1);
    const startWeekday = firstDay.getDay();
    const daysInMonth = new Date(viewYear, viewMonth, 0).getDate();
    const slots: Array<number | null> = [];
    for (let i = 0; i < startWeekday; i += 1) slots.push(null);
    for (let day = 1; day <= daysInMonth; day += 1) slots.push(day);
    while (slots.length % 7 !== 0) slots.push(null);
    return slots;
  }, [viewYear, viewMonth]);

  const [hourInput, setHourInput] = useState("");
  const [minuteInput, setMinuteInput] = useState("");

  useEffect(() => {
    setHourInput(selected ? pad2(selected.hour) : "");
    setMinuteInput(selected ? pad2(selected.minute) : "");
  }, [selected]);

  const title = `${monthNames[viewMonth - 1]} de ${viewYear}`;
  const displayValue = formatDisplay(selected);
  const timePlaceholder = "HH:MM";

  function normalizeTimeInput(raw: string, max: number) {
    const digits = raw.replace(/\D/g, "").slice(0, 2);
    if (!digits) return "";
    const value = clamp(Number(digits), 0, max);
    return pad2(value);
  }

  function applyTimeChange(nextHour: string, nextMinute: string) {
    if (!nextHour && !nextMinute) return;
    const hourValue = nextHour ? Number(nextHour) : (selected?.hour ?? 9);
    const minuteValue = nextMinute ? Number(nextMinute) : (selected?.minute ?? 0);
    updateParts({ hour: clamp(hourValue, 0, 23), minute: clamp(minuteValue, 0, 59) });
  }

  return (
    <div className="create-date-field" onBlur={handleBlur}>
      <button
        id={props.id}
        type="button"
        className="create-date-input"
        onClick={() => setOpen((prev) => !prev)}
      >
        <span className={`create-date-value${displayValue ? "" : " is-placeholder"}`}>
          {displayValue || props.placeholder || "dd/mm/aaaa --:--"}
        </span>
        <span className="create-date-icon" aria-hidden>
          <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
            <rect x="3" y="4.5" width="14" height="12" rx="2" />
            <path d="M6 3v3M14 3v3M3 8h14" />
          </svg>
        </span>
      </button>
      {open ? (
        <div className="create-date-popover" role="dialog" aria-label="Selecionar data e hora">
          <div className="create-date-header">
            <button
              type="button"
              className="create-date-nav"
              onClick={() => {
                const nextMonth = viewMonth - 1;
                if (nextMonth < 1) {
                  setViewMonth(12);
                  setViewYear((current) => current - 1);
                } else {
                  setViewMonth(nextMonth);
                }
              }}
              aria-label="Mes anterior"
            >
              <span aria-hidden>‹</span>
            </button>
            <div className="create-date-title">{title}</div>
            <button
              type="button"
              className="create-date-nav"
              onClick={() => {
                const nextMonth = viewMonth + 1;
                if (nextMonth > 12) {
                  setViewMonth(1);
                  setViewYear((current) => current + 1);
                } else {
                  setViewMonth(nextMonth);
                }
              }}
              aria-label="Mes seguinte"
            >
              <span aria-hidden>›</span>
            </button>
          </div>
          <div className="create-date-weekdays">
            {["D", "S", "T", "Q", "Q", "S", "S"].map((label) => (
              <span key={label}>{label}</span>
            ))}
          </div>
          <div className="create-date-grid">
            {daysGrid.map((day, index) => {
              if (!day) return <span key={`empty-${index}`} className="create-date-empty" />;
              const isSelected = selected?.year === viewYear && selected?.month === viewMonth && selected?.day === day;
              return (
                <button
                  key={`${viewYear}-${viewMonth}-${day}`}
                  type="button"
                  className={`create-date-day${isSelected ? " is-selected" : ""}`}
                  onClick={() => updateParts({ year: viewYear, month: viewMonth, day })}
                >
                  {day}
                </button>
              );
            })}
          </div>
          <div className="create-date-time">
            <div className="create-date-time-group">
              <span>Hora</span>
              <div className="create-date-time-input">
                <input
                  className="create-date-time-text"
                  inputMode="numeric"
                  placeholder={timePlaceholder}
                  value={hourInput}
                  onChange={(event) => setHourInput(event.target.value.replace(/\D/g, "").slice(0, 2))}
                  onBlur={(event) => {
                    const normalized = normalizeTimeInput(event.target.value, 23);
                    setHourInput(normalized);
                    applyTimeChange(normalized, minuteInput);
                  }}
                  aria-label="Hora"
                />
                <span className="create-date-time-sep">:</span>
                <input
                  className="create-date-time-text"
                  inputMode="numeric"
                  placeholder={timePlaceholder}
                  value={minuteInput}
                  onChange={(event) => setMinuteInput(event.target.value.replace(/\D/g, "").slice(0, 2))}
                  onBlur={(event) => {
                    const normalized = normalizeTimeInput(event.target.value, 59);
                    setMinuteInput(normalized);
                    applyTimeChange(hourInput, normalized);
                  }}
                  aria-label="Minutos"
                />
              </div>
            </div>
            <div className="create-date-actions">
              <button type="button" className="ghost-button" onClick={() => props.onChange("")}>
                Limpar
              </button>
              <button
                type="button"
                className="ghost-button"
                onClick={() => {
                  const now = new Date();
                  updateParts({
                    year: now.getFullYear(),
                    month: now.getMonth() + 1,
                    day: now.getDate(),
                    hour: now.getHours(),
                    minute: now.getMinutes() - (now.getMinutes() % 5),
                  });
                }}
              >
                Hoje
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
