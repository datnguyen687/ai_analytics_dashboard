"use client";

import { ReactNode } from "react";

export function Card({
  title,
  subtitle,
  action,
  children,
  className = "",
}: {
  title?: string;
  subtitle?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`card p-4 sm:p-5 ${className}`}>
      {(title || action) && (
        <header className="mb-4 flex items-start justify-between gap-3">
          <div>
            {title && (
              <h2 className="text-sm font-semibold tracking-tight text-[var(--text-primary)]">
                {title}
              </h2>
            )}
            {subtitle && (
              <p className="mt-0.5 text-xs text-[var(--muted)]">{subtitle}</p>
            )}
          </div>
          {action}
        </header>
      )}
      {children}
    </section>
  );
}

export function StatTile({
  label,
  value,
  hint,
  color = "var(--series-1)",
  delta,
  chart,
}: {
  label: string;
  value: string;
  hint?: string;
  /** Accent color — one categorical/status slot per tile, assigned by the caller. */
  color?: string;
  delta?: { value: string; direction: "up" | "down"; good: boolean };
  chart?: ReactNode;
}) {
  return (
    <div className="card relative overflow-hidden p-4">
      <span
        aria-hidden
        className="absolute inset-y-0 left-0 w-[3px]"
        style={{ background: color }}
      />
      <div className="flex items-start justify-between gap-2">
        <p className="text-xs font-medium text-[var(--text-secondary)]">{label}</p>
        {delta && (
          <span
            className="flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px] font-medium"
            style={{
              color: delta.good ? "var(--status-good)" : "var(--status-critical)",
              background: delta.good
                ? "color-mix(in srgb, var(--status-good) 12%, transparent)"
                : "color-mix(in srgb, var(--status-critical) 12%, transparent)",
            }}
          >
            <span aria-hidden>{delta.direction === "up" ? "▲" : "▼"}</span>
            {delta.value}
          </span>
        )}
      </div>
      <p className="mt-2 text-2xl font-semibold tracking-tight text-[var(--text-primary)]">
        {value}
      </p>
      {hint && <p className="mt-1 text-[11px] text-[var(--muted)]">{hint}</p>}
      {chart && <div className="-mx-1 mt-2">{chart}</div>}
    </div>
  );
}

/** Horizontal share bar — used inline in tables to make magnitude scannable. */
export function MiniBar({
  value,
  max,
  color = "var(--series-1)",
}: {
  value: number;
  max: number;
  color?: string;
}) {
  const pct = max === 0 ? 0 : Math.max(2, (value / max) * 100);
  return (
    <span className="flex items-center gap-2">
      <span className="h-1.5 w-16 overflow-hidden rounded-full bg-[var(--hover)]">
        <span className="block h-full rounded-full" style={{ width: `${pct}%`, background: color }} />
      </span>
      <span className="tnum">{value.toLocaleString()}</span>
    </span>
  );
}

const STATUS_TONE: Record<string, { color: string; label: string }> = {
  delivered: { color: "var(--status-good)", label: "Delivered" },
  delayed: { color: "var(--status-critical)", label: "Delayed" },
  in_transit: { color: "var(--series-1)", label: "In transit" },
  exception: { color: "var(--status-serious)", label: "Exception" },
  canceled: { color: "var(--muted)", label: "Canceled" },
};

export function StatusChip({ status }: { status: string }) {
  const t = STATUS_TONE[status] ?? { color: "var(--muted)", label: status };
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium"
      style={{
        color: t.color,
        background: `color-mix(in srgb, ${t.color} 12%, transparent)`,
      }}
    >
      <span aria-hidden className="size-1.5 rounded-full" style={{ background: t.color }} />
      {t.label}
    </span>
  );
}

export function Pill({
  children,
  active,
  onClick,
}: {
  children: ReactNode;
  active?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full border px-3 py-1 text-xs transition-colors ${
        active
          ? "border-transparent bg-[var(--series-1)] text-white"
          : "border-[var(--border)] text-[var(--text-secondary)] hover:bg-[var(--hover)]"
      }`}
    >
      {children}
    </button>
  );
}

export function DataTable({
  columns,
  rows,
  maxHeight = 320,
}: {
  columns: string[];
  rows: (string | number)[][];
  maxHeight?: number;
}) {
  return (
    <div className="overflow-auto rounded-lg border border-[var(--border)]" style={{ maxHeight }}>
      <table className="w-full min-w-max text-left text-xs tnum">
        <thead className="sticky top-0 bg-[var(--surface-1)]">
          <tr className="border-b border-[var(--border)]">
            {columns.map((c) => (
              <th
                key={c}
                className="px-3 py-2 font-medium whitespace-nowrap text-[var(--text-secondary)]"
              >
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-b border-[var(--border)] last:border-0">
              {r.map((cell, j) => (
                <td
                  key={j}
                  className="px-3 py-1.5 whitespace-nowrap text-[var(--text-primary)]"
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function Badge({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: "neutral" | "good" | "warning" | "critical";
}) {
  const map = {
    neutral: "var(--muted)",
    good: "var(--status-good)",
    warning: "var(--status-warning)",
    critical: "var(--status-critical)",
  } as const;
  return (
    <span className="inline-flex items-center gap-1.5 rounded-md border border-[var(--border)] px-2 py-0.5 text-[11px] text-[var(--text-secondary)]">
      <span aria-hidden className="size-1.5 rounded-full" style={{ background: map[tone] }} />
      {children}
    </span>
  );
}
