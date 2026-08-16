/**
 * Chart colors are referenced as CSS custom properties so light/dark swap in a
 * single place (see globals.css). SVG attributes accept var() directly.
 */
export const COLORS = {
  primary: "var(--series-1)",
  accent: "var(--series-2)",
  aqua: "var(--series-3)",
  violet: "var(--series-7)",
  success: "var(--status-good)",
  danger: "var(--status-critical)",
  warning: "var(--status-warning)",
  grid: "var(--gridline)",
  axis: "var(--muted)",
  surface: "var(--surface-1)",
} as const;

/** Fixed categorical order — assigned by slot, never cycled. */
export const CATEGORICAL = [
  "var(--series-1)",
  "var(--series-2)",
  "var(--series-3)",
  "var(--series-4)",
  "var(--series-5)",
  "var(--series-6)",
  "var(--series-7)",
  "var(--series-8)",
];
