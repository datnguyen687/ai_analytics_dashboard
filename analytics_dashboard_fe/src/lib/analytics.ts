import type { Filters } from "./types";

/**
 * Default filters: empty date range means "all dates" — the backend treats
 * missing from/to as no restriction, so this returns the full dataset.
 */
export const DEFAULT_FILTERS: Filters = {
  from: "",
  to: "",
  regions: [],
  carriers: [],
  categories: [],
};

/** Advance a YYYY-MM bucket by delta months. Used to build date-range presets. */
export function addMonths(bucket: string, delta: number): string {
  const [y, m] = bucket.split("-").map(Number);
  const d = new Date(Date.UTC(y, m - 1 + delta, 1));
  return d.toISOString().slice(0, 7);
}

/** Format a YYYY-MM bucket as "Jan 25". */
export function fmtMonth(bucket: string): string {
  const [y, m] = bucket.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, 1)).toLocaleString("en-US", {
    month: "short",
    year: "2-digit",
    timeZone: "UTC",
  });
}

export function fmtPct(x: number, digits = 1): string {
  return `${(x * 100).toFixed(digits)}%`;
}

export function fmtMoney(x: number): string {
  return `$${Math.round(x).toLocaleString("en-US")}`;
}
