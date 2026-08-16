"use client";

import { ReactNode, useEffect, useRef, useState } from "react";

function useClickOutside(onClose: () => void) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    const esc = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("mousedown", handler);
    document.addEventListener("keydown", esc);
    return () => {
      document.removeEventListener("mousedown", handler);
      document.removeEventListener("keydown", esc);
    };
  }, [onClose]);
  return ref;
}

function Chevron({ open }: { open: boolean }) {
  return (
    <svg
      viewBox="0 0 16 16"
      aria-hidden
      className={`size-3.5 shrink-0 text-[var(--muted)] transition-transform ${open ? "rotate-180" : ""}`}
    >
      <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function Check() {
  return (
    <svg viewBox="0 0 16 16" aria-hidden className="size-3.5">
      <path d="M3 8.5l3.2 3.2L13 5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function Popover({
  label,
  summary,
  count,
  children,
  width = 240,
}: {
  label: string;
  summary: string;
  count?: number;
  children: (close: () => void) => ReactNode;
  width?: number;
}) {
  const [open, setOpen] = useState(false);
  const ref = useClickOutside(() => setOpen(false));

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex h-9 min-w-[9.5rem] items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--surface-1)] px-3 text-left text-xs transition-colors hover:bg-[var(--hover)]"
      >
        <span className="text-[var(--muted)]">{label}</span>
        <span className="ml-auto flex items-center gap-1.5 truncate font-medium text-[var(--text-primary)]">
          <span className="truncate">{summary}</span>
          {!!count && (
            <span
              className="rounded-full px-1.5 text-[10px] font-semibold text-white"
              style={{ background: "var(--series-1)" }}
            >
              {count}
            </span>
          )}
        </span>
        <Chevron open={open} />
      </button>

      {open && (
        <div
          className="absolute left-0 top-[calc(100%+6px)] z-40 rounded-xl border border-[var(--border)] bg-[var(--surface-1)] p-2 shadow-xl shadow-black/10"
          style={{ width }}
        >
          {children(() => setOpen(false))}
        </div>
      )}
    </div>
  );
}

export function MultiSelect({
  label,
  options,
  selected,
  onChange,
  colors,
}: {
  label: string;
  options: string[];
  selected: string[];
  onChange: (next: string[]) => void;
  colors?: string[];
}) {
  const summary = selected.length === 0 ? "All" : selected.length === 1 ? selected[0] : `${selected.length} selected`;

  const toggle = (v: string) =>
    onChange(selected.includes(v) ? selected.filter((x) => x !== v) : [...selected, v]);

  return (
    <Popover label={label} summary={summary} count={selected.length > 1 ? selected.length : 0}>
      {() => (
        <>
          <div className="max-h-64 overflow-auto">
            {options.map((o, i) => {
              const on = selected.includes(o);
              return (
                <button
                  key={o}
                  type="button"
                  onClick={() => toggle(o)}
                  className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-xs transition-colors hover:bg-[var(--hover)]"
                >
                  <span
                    className="grid size-4 shrink-0 place-items-center rounded border text-white"
                    style={{
                      borderColor: on ? "transparent" : "var(--border)",
                      background: on ? (colors?.[i % colors.length] ?? "var(--series-1)") : "transparent",
                    }}
                  >
                    {on && <Check />}
                  </span>
                  {colors && (
                    <span aria-hidden className="size-2 rounded-sm" style={{ background: colors[i % colors.length] }} />
                  )}
                  <span className="truncate text-[var(--text-primary)]">{o}</span>
                </button>
              );
            })}
          </div>
          {selected.length > 0 && (
            <button
              type="button"
              onClick={() => onChange([])}
              className="mt-1 w-full border-t border-[var(--border)] pt-2 text-xs text-[var(--series-1)]"
            >
              Clear {label.toLowerCase()}
            </button>
          )}
        </>
      )}
    </Popover>
  );
}

export function Select({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: { value: string; label: string }[];
  value: string;
  onChange: (v: string) => void;
}) {
  const current = options.find((o) => o.value === value)?.label ?? value;
  return (
    <Popover label={label} summary={current} width={200}>
      {(close) => (
        <div className="max-h-64 overflow-auto">
          {options.map((o) => (
            <button
              key={o.value}
              type="button"
              onClick={() => {
                onChange(o.value);
                close();
              }}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-xs transition-colors hover:bg-[var(--hover)]"
            >
              <span className="w-4 text-[var(--series-1)]">{o.value === value && <Check />}</span>
              <span className="text-[var(--text-primary)]">{o.label}</span>
            </button>
          ))}
        </div>
      )}
    </Popover>
  );
}

export interface RangePreset {
  label: string;
  from: string;
  to: string;
}

export function DateRangePicker({
  from,
  to,
  min,
  max,
  presets,
  onChange,
}: {
  from: string;
  to: string;
  min: string;
  max: string;
  presets: RangePreset[];
  onChange: (from: string, to: string) => void;
}) {
  const match = presets.find((p) => p.from === from && p.to === to);
  const summary = match ? match.label : from && to ? `${from} → ${to}` : "All dates";

  return (
    <Popover label="Period" summary={summary} width={278}>
      {(close) => (
        <>
          <div className="flex flex-col">
            {presets.map((p) => {
              const on = p.from === from && p.to === to;
              return (
                <button
                  key={p.label}
                  type="button"
                  onClick={() => {
                    onChange(p.from, p.to);
                    close();
                  }}
                  className="flex items-center justify-between rounded-lg px-2 py-1.5 text-left text-xs transition-colors hover:bg-[var(--hover)]"
                >
                  <span className="text-[var(--text-primary)]">{p.label}</span>
                  <span className="text-[var(--series-1)]">{on && <Check />}</span>
                </button>
              );
            })}
          </div>
          <div className="mt-2 border-t border-[var(--border)] pt-2">
            <p className="mb-1.5 px-2 text-[10px] uppercase tracking-wide text-[var(--muted)]">
              Custom range
            </p>
            <div className="flex items-center gap-2 px-2 pb-1">
              <input
                type="date"
                value={from}
                min={min}
                max={to}
                onChange={(e) => onChange(e.target.value, to)}
                className="w-full rounded-md border border-[var(--border)] bg-transparent px-2 py-1 text-[11px] tnum text-[var(--text-primary)]"
              />
              <span className="text-[var(--muted)]">–</span>
              <input
                type="date"
                value={to}
                min={from}
                max={max}
                onChange={(e) => onChange(from, e.target.value)}
                className="w-full rounded-md border border-[var(--border)] bg-transparent px-2 py-1 text-[11px] tnum text-[var(--text-primary)]"
              />
            </div>
          </div>
        </>
      )}
    </Popover>
  );
}

export function SearchInput({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
}) {
  return (
    <label className="flex h-9 flex-1 min-w-[12rem] items-center gap-2 rounded-lg border border-[var(--border)] bg-[var(--surface-1)] px-3">
      <svg viewBox="0 0 16 16" aria-hidden className="size-3.5 text-[var(--muted)]">
        <circle cx="7" cy="7" r="4.5" fill="none" stroke="currentColor" strokeWidth="1.6" />
        <path d="M10.5 10.5L14 14" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
      </svg>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full bg-transparent text-xs outline-none placeholder:text-[var(--muted)]"
      />
      {value && (
        <button type="button" onClick={() => onChange("")} className="text-[11px] text-[var(--muted)]">
          ✕
        </button>
      )}
    </label>
  );
}
